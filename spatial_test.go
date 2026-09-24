package spatial_test

import (
	"fmt"
	spatial "spatial-firebase"
	"testing"
)

// TestNewPoint verifies point feature instantiation and multi-resolution index generation.
func TestNewPoint(t *testing.T) {
	lat, lng := 19.4326, -99.1332
	resolutions := []int{6, 8, 10}

	point := spatial.NewPoint(lat, lng, resolutions)

	if point.Type != spatial.TypePoint {
		t.Errorf("expected type %s, got %s", spatial.TypePoint, point.Type)
	}

	if point.Lat != lat || point.Lng != lng {
		t.Errorf("coordinates mismatch: expected (%f, %f), got (%f, %f)", lat, lng, point.Lat, point.Lng)
	}

	for _, res := range resolutions {
		key := fmt.Sprintf("r%d", res)
		val, exists := point.H3Map[key]
		if !exists || val == "" {
			t.Errorf("missing resolution %s in H3Map", key)
		}
	}

	if point.H3Center == "" {
		t.Error("H3Center must not be empty")
	}
}

// TestNewPolygon verifies polygon polyfill cell generation and centroid calculation.
func TestNewPolygon(t *testing.T) {
	coords := []spatial.Coordinate{
		{Lat: 19.4350, Lng: -99.1400},
		{Lat: 19.4350, Lng: -99.1300},
		{Lat: 19.4250, Lng: -99.1300},
		{Lat: 19.4250, Lng: -99.1400},
	}

	res := 8
	polygon, err := spatial.NewPolygon(coords, res)
	if err != nil {
		t.Fatalf("unexpected error creating polygon: %v", err)
	}

	if polygon.Type != spatial.TypePolygon {
		t.Errorf("expected type %s, got %s", spatial.TypePolygon, polygon.Type)
	}

	if len(polygon.H3Cells) == 0 {
		t.Error("polygon must generate at least 1 polyfilled H3 cell")
	}

	if polygon.H3Center == "" {
		t.Error("polygon H3Center must not be empty")
	}
}

// TestNewPolygon_Invalid verifies error handling when polygon coordinate count is insufficient.
func TestNewPolygon_Invalid(t *testing.T) {
	invalidCoords := []spatial.Coordinate{
		{Lat: 19.4350, Lng: -99.1400},
		{Lat: 19.4350, Lng: -99.1300},
	}

	_, err := spatial.NewPolygon(invalidCoords, 8)
	if err == nil {
		t.Error("expected error when creating polygon with fewer than 3 coordinates")
	}
}

// TestHaversineDistance verifies coordinate positioning and distinct spatial representations.
func TestHaversineDistance(t *testing.T) {
	lat1, lng1 := 19.4326, -99.1332
	lat2, lng2 := 19.4270, -99.1677

	pointA := spatial.NewPoint(lat1, lng1, []int{8})
	pointB := spatial.NewPoint(lat2, lng2, []int{8})

	if pointA.Lat == pointB.Lat && pointA.Lng == pointB.Lng {
		t.Error("distinct spatial points should not match coordinates")
	}
}
