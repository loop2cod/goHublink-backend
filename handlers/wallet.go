package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

const (
	WalletTierBronze = "bronze"
	WalletTierSilver = "silver"
	WalletTierGold   = "gold"

	WalletTxTypeCredit = "credit"
	WalletTxTypeDebit  = "debit"
)

func walletTierForBalance(balance int) string {
	switch {
	case balance >= 5000:
		return WalletTierGold
	case balance >= 1000:
		return WalletTierSilver
	default:
		return WalletTierBronze
	}
}

func getWalletForPhoneNumber(tx *gorm.DB, phoneNumber string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := tx.Where("phone_number = ?", phoneNumber).First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	wallet = models.Wallet{
		PhoneNumber: phoneNumber,
		Balance:     0,
		Tier:        WalletTierBronze,
	}
	if err := tx.Create(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func applyWalletTier(tx *gorm.DB, wallet *models.Wallet) error {
	tier := walletTierForBalance(wallet.Balance)
	if wallet.Tier == tier {
		return nil
	}
	wallet.Tier = tier
	return tx.Model(wallet).Update("tier", tier).Error
}

func GetWallet(c *gin.Context) {
	phoneNumber, exists := c.Get("phone_number")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	phone := phoneNumber.(string)

	var wallet models.Wallet
	err := db.DB.Where("phone_number = ?", phone).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wallet = models.Wallet{
				PhoneNumber: phone,
				Balance:     0,
				Tier:        WalletTierBronze,
			}
			if err := db.DB.Create(&wallet).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, wallet)
}

func GetWalletTransactions(c *gin.Context) {
	phoneNumber, exists := c.Get("phone_number")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	phone := phoneNumber.(string)

	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	query := db.DB.Model(&models.WalletTransaction{}).
		Where("phone_number = ?", phone).
		Order("created_at DESC")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var transactions []models.WalletTransaction
	if err := query.
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
		"page":         page,
		"limit":        limit,
		"total_pages":  totalPages,
	})
}

type creditWalletRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Amount      int    `json:"amount" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	ReferenceID string `json:"reference_id"`
}

func CreditWallet(c *gin.Context) {
	var req creditWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	var wallet *models.Wallet
	var txErr error

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		wallet, txErr = getWalletForPhoneNumber(tx, req.PhoneNumber)
		if txErr != nil {
			return txErr
		}

		wallet.Balance += req.Amount
		if txErr := tx.Model(wallet).Update("balance", wallet.Balance).Error; txErr != nil {
			return txErr
		}
		if txErr := applyWalletTier(tx, wallet); txErr != nil {
			return txErr
		}

		transaction := models.WalletTransaction{
			PhoneNumber: req.PhoneNumber,
			Amount:      req.Amount,
			Type:        WalletTxTypeCredit,
			Reason:      req.Reason,
			ReferenceID: req.ReferenceID,
		}
		if txErr := tx.Create(&transaction).Error; txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":      wallet,
		"transaction": wallet.Balance,
	})
}

type debitWalletRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Amount      int    `json:"amount" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	ReferenceID string `json:"reference_id"`
}

func DebitWallet(c *gin.Context) {
	var req debitWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	var wallet *models.Wallet
	var transaction *models.WalletTransaction

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		wallet, txErr = getWalletForPhoneNumber(tx, req.PhoneNumber)
		if txErr != nil {
			return txErr
		}

		if wallet.Balance < req.Amount {
			return errors.New("insufficient balance")
		}

		wallet.Balance -= req.Amount
		if txErr := tx.Model(wallet).Update("balance", wallet.Balance).Error; txErr != nil {
			return txErr
		}
		if txErr := applyWalletTier(tx, wallet); txErr != nil {
			return txErr
		}

		transaction = &models.WalletTransaction{
			PhoneNumber: req.PhoneNumber,
			Amount:      req.Amount,
			Type:        WalletTxTypeDebit,
			Reason:      req.Reason,
			ReferenceID: req.ReferenceID,
		}
		if txErr := tx.Create(transaction).Error; txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		if err.Error() == "insufficient balance" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":      wallet,
		"transaction": transaction,
	})
}

// awardPointsToWallet is a helper used by other handlers (e.g. scan matching)
// to credit a customer's wallet atomically and return the updated wallet.
func awardPointsToWallet(tx *gorm.DB, phoneNumber string, amount int, reason, referenceID string) (*models.Wallet, error) {
	wallet, err := getWalletForPhoneNumber(tx, phoneNumber)
	if err != nil {
		return nil, err
	}

	wallet.Balance += amount
	if err := tx.Model(wallet).Update("balance", wallet.Balance).Error; err != nil {
		return nil, err
	}
	if err := applyWalletTier(tx, wallet); err != nil {
		return nil, err
	}

	transaction := models.WalletTransaction{
		PhoneNumber: phoneNumber,
		Amount:      amount,
		Type:        WalletTxTypeCredit,
		Reason:      reason,
		ReferenceID: referenceID,
	}
	if err := tx.Create(&transaction).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}
