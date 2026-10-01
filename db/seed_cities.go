package db

import (
	"encoding/json"
	"log"

	"gohublink/backend/models"
)

func SeedCities() error {
	var cityCount int64
	if err := DB.Model(&models.City{}).Count(&cityCount).Error; err != nil {
		return err
	}
	if cityCount > 0 {
		return nil
	}

	blrCity := models.City{
		ID:              "bengaluru",
		Slug:            "bengaluru",
		Name:            "Bengaluru",
		Region:          "South India",
		State:           "Karnataka",
		Country:         "India",
		Latitude:        12.9716,
		Longitude:       77.5946,
		BoundaryGeoJSON: getBengaluruBoundaryJSON(),
		IsActive:        true,
	}

	if err := DB.Create(&blrCity).Error; err != nil {
		return err
	}

	// The exact 9 divisions shown in the reference image:
	// West, Central, Indiranagar, Koramangala, Jayanagar, ORR Bellandur, Whitefield, South Outer, North Bangalore Airport
	divisions := []models.Division{
		{
			ID:             "blr-west",
			CityID:         "bengaluru",
			Name:           "West",
			Code:           "BLR-WST",
			CenterLat:      12.980,
			CenterLng:      77.520,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{13.080, 77.520},
				{13.040, 77.560},
				{12.970, 77.570},
				{12.930, 77.550},
				{12.910, 77.450},
				{12.960, 77.460},
				{13.020, 77.500},
				{13.080, 77.520},
			}),
			Color:     "green",
			IsActive:  true,
			SortOrder: 1,
		},
		{
			ID:             "blr-central",
			CityID:         "bengaluru",
			Name:           "Central",
			Code:           "BLR-CEN",
			CenterLat:      12.975,
			CenterLng:      77.595,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{13.040, 77.560},
				{13.040, 77.625},
				{12.960, 77.625},
				{12.960, 77.570},
				{12.970, 77.570},
				{13.040, 77.560},
			}),
			Color:     "orange",
			IsActive:  true,
			SortOrder: 2,
		},
		{
			ID:             "blr-indiranagar",
			CityID:         "bengaluru",
			Name:           "Indiranagar",
			Code:           "BLR-IND",
			CenterLat:      12.985,
			CenterLng:      77.650,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{13.040, 77.625},
				{13.040, 77.695},
				{12.960, 77.695},
				{12.960, 77.625},
				{13.040, 77.625},
			}),
			Color:     "yellow",
			IsActive:  true,
			SortOrder: 3,
		},
		{
			ID:             "blr-koramangala",
			CityID:         "bengaluru",
			Name:           "Koramangala",
			Code:           "BLR-KOR",
			CenterLat:      12.935,
			CenterLng:      77.625,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{12.960, 77.570},
				{12.960, 77.625},
				{12.960, 77.695},
				{12.920, 77.690},
				{12.900, 77.640},
				{12.900, 77.590},
				{12.930, 77.550},
				{12.960, 77.570},
			}),
			Color:     "purple",
			IsActive:  true,
			SortOrder: 4,
		},
		{
			ID:             "blr-jayanagar",
			CityID:         "bengaluru",
			Name:           "Jayanagar",
			Code:           "BLR-JAY",
			CenterLat:      12.925,
			CenterLng:      77.575,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{12.930, 77.550},
				{12.900, 77.590},
				{12.860, 77.600},
				{12.840, 77.560},
				{12.860, 77.500},
				{12.910, 77.450},
				{12.930, 77.550},
			}),
			Color:     "pink",
			IsActive:  true,
			SortOrder: 5,
		},
		{
			ID:             "blr-orr-bellandur",
			CityID:         "bengaluru",
			Name:           "ORR, Bellandur",
			Code:           "BLR-ORR",
			CenterLat:      12.925,
			CenterLng:      77.720,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{12.960, 77.695},
				{12.980, 77.800},
				{12.900, 77.760},
				{12.860, 77.720},
				{12.860, 77.660},
				{12.900, 77.640},
				{12.920, 77.690},
				{12.960, 77.695},
			}),
			Color:     "blue",
			IsActive:  true,
			SortOrder: 6,
		},
		{
			ID:             "blr-whitefield",
			CityID:         "bengaluru",
			Name:           "Whitefield",
			Code:           "BLR-WHT",
			CenterLat:      13.000,
			CenterLng:      77.750,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{13.040, 77.695},
				{13.080, 77.800},
				{12.980, 77.800},
				{12.960, 77.695},
				{13.040, 77.695},
			}),
			Color:     "teal",
			IsActive:  true,
			SortOrder: 7,
		},
		{
			ID:             "blr-south-outer",
			CityID:         "bengaluru",
			Name:           "South Outer",
			Code:           "BLR-SOU",
			CenterLat:      12.830,
			CenterLng:      77.600,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{12.860, 77.500},
				{12.840, 77.560},
				{12.860, 77.600},
				{12.900, 77.640},
				{12.860, 77.660},
				{12.860, 77.720},
				{12.820, 77.720},
				{12.780, 77.640},
				{12.780, 77.520},
				{12.820, 77.480},
				{12.860, 77.500},
			}),
			Color:     "cyan",
			IsActive:  true,
			SortOrder: 8,
		},
		{
			ID:             "blr-north-airport",
			CityID:         "bengaluru",
			Name:           "North Bangalore, Airport",
			Code:           "BLR-NTH",
			CenterLat:      13.120,
			CenterLng:      77.640,
			PolygonGeoJSON: getPolygonJSON([][]float64{
				{13.040, 77.560},
				{13.080, 77.520},
				{13.150, 77.560},
				{13.220, 77.580},
				{13.250, 77.650},
				{13.250, 77.730},
				{13.200, 77.780},
				{13.080, 77.800},
				{13.040, 77.695},
				{13.040, 77.625},
				{13.040, 77.560},
			}),
			Color:     "lavender",
			IsActive:  true,
			SortOrder: 9,
		},
	}

	for _, div := range divisions {
		if err := DB.Create(&div).Error; err != nil {
			return err
		}
	}

	log.Printf("Seeded default city Bengaluru with exact %d divisions from reference image", len(divisions))
	return nil
}

func getPolygonJSON(coords [][]float64) string {
	b, _ := json.Marshal(coords)
	return string(b)
}

func getBengaluruBoundaryJSON() string {
	coords := [][]float64{
		{12.780, 77.520},
		{12.820, 77.480},
		{12.910, 77.450},
		{12.960, 77.460},
		{13.020, 77.500},
		{13.080, 77.520},
		{13.150, 77.560},
		{13.220, 77.580},
		{13.250, 77.650},
		{13.250, 77.730},
		{13.200, 77.780},
		{13.080, 77.800},
		{12.980, 77.800},
		{12.900, 77.760},
		{12.820, 77.720},
		{12.780, 77.640},
		{12.780, 77.520},
	}
	b, _ := json.Marshal(coords)
	return string(b)
}
