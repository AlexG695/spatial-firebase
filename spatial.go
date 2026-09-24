package spatial

import (
	"fmt"

	"github.com/uber/h3-go/v3"
)

// GeometryType defines supported spatial geometry primitives.
type GeometryType string

const (
	TypePoint   GeometryType = "Point"
	TypePolygon GeometryType = "Polygon"
)

// SpatialFeature represents indexed geospatial metadata stored in Firestore documents.
type SpatialFeature struct {
	Type GeometryType `firestore:"type" json:"type"`
	Lat  float64      `firestore:"lat,omitempty" json:"lat,omitempty"`
	Lng  float64      `firestore:"lng,omitempty" json:"lng,omitempty"`

	H3Map    map[string]string `firestore:"h3_map,omitempty" json:"h3_map,omitempty"`
	H3Cells  []string          `firestore:"h3_cells,omitempty" json:"h3_cells,omitempty"`
	H3Center string            `firestore:"h3_center,omitempty" json:"h3_center,omitempty"`
}

// Coordinate represents a WGS84 geographic coordinate pair.
type Coordinate struct {
	Lat float64
	Lng float64
}

// NewPoint constructs a SpatialFeature for a point, precomputing H3 indices for specified resolutions.
func NewPoint(lat, lng float64, resolutions []int) SpatialFeature {
	coord := h3.GeoCoord{Latitude: lat, Longitude: lng}
	h3Map := make(map[string]string)

	var centerRes8 string

	for _, res := range resolutions {
		index := h3.FromGeo(coord, res)
		hexStr := fmt.Sprintf("%x", index)
		key := fmt.Sprintf("r%d", res)
		h3Map[key] = hexStr

		if res == 8 {
			centerRes8 = hexStr
		}
	}

	// Default centroid to resolution 8 if not explicitly present in resolutions.
	if centerRes8 == "" {
		centerRes8 = fmt.Sprintf("%x", h3.FromGeo(coord, 8))
	}

	return SpatialFeature{
		Type:     TypePoint,
		Lat:      lat,
		Lng:      lng,
		H3Map:    h3Map,
		H3Center: centerRes8,
	}
}

// NewPolygon generates a SpatialFeature covering a polygon boundary via H3 polyfill at the specified resolution.
func NewPolygon(coordinates []Coordinate, resolution int) (SpatialFeature, error) {
	if len(coordinates) < 3 {
		return SpatialFeature{}, fmt.Errorf("polygon requires at least 3 coordinates")
	}

	geoCoords := make([]h3.GeoCoord, len(coordinates))
	var sumLat, sumLng float64

	for i, c := range coordinates {
		geoCoords[i] = h3.GeoCoord{Latitude: c.Lat, Longitude: c.Lng}
		sumLat += c.Lat
		sumLng += c.Lng
	}

	centerLat := sumLat / float64(len(coordinates))
	centerLng := sumLng / float64(len(coordinates))
	centerIndex := fmt.Sprintf("%x", h3.FromGeo(h3.GeoCoord{Latitude: centerLat, Longitude: centerLng}, resolution))

	polygon := h3.GeoPolygon{
		Geofence: geoCoords,
	}

	cells := h3.Polyfill(polygon, resolution)
	cellStrings := make([]string, len(cells))

	for i, cell := range cells {
		cellStrings[i] = fmt.Sprintf("%x", cell)
	}

	return SpatialFeature{
		Type:     TypePolygon,
		Lat:      centerLat,
		Lng:      centerLng,
		H3Cells:  cellStrings,
		H3Center: centerIndex,
	}, nil
}
