package spatial

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/firestore"
	"github.com/uber/h3-go/v3"
)

// SearchOptions defines optional parameters for spatial queries.
type SearchOptions struct {
	Limit      int // Maximum number of documents to return per query batch.
	Resolution int // H3 resolution level to query (defaults to 8).
}

// h3EdgeLengthsKm maps H3 resolutions to their approximate hexagon edge length in kilometers.
var h3EdgeLengthsKm = map[int]float64{
	0: 1107.71, 1: 418.67, 2: 158.24, 3: 59.81,
	4: 22.61, 5: 8.54, 6: 3.23, 7: 1.22,
	8: 0.46, 9: 0.17, 10: 0.065,
}

// haversineDistance computes the great-circle distance between two geographic coordinates in kilometers.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// SearchRadius retrieves documents within radiusKm of (centerLat, centerLng) using H3 k-ring expansion and Haversine post-filtering.
func (c *Client) SearchRadius(ctx context.Context, collection string, centerLat, centerLng float64, radiusKm float64, opts SearchOptions) ([]*firestore.DocumentSnapshot, error) {
	if opts.Resolution == 0 {
		opts.Resolution = 8
	}

	centerCoord := h3.GeoCoord{Latitude: centerLat, Longitude: centerLng}
	centerCell := h3.FromGeo(centerCoord, opts.Resolution)

	// Approximate k-ring radius to guarantee coverage of the target circular radius.
	edgeLengthKm, ok := h3EdgeLengthsKm[opts.Resolution]
	if !ok {
		edgeLengthKm = 0.46
	}

	kRing := int(math.Ceil(radiusKm / edgeLengthKm))
	if kRing < 1 {
		kRing = 1
	}

	cells := h3.KRing(centerCell, kRing)
	targetCells := make([]string, len(cells))
	for i, cell := range cells {
		targetCells[i] = fmt.Sprintf("%x", cell)
	}

	// Chunk cell queries into batches of up to 30 elements to respect Firestore 'in' query constraints.
	visitedDocs := make(map[string]bool)
	var allDocs []*firestore.DocumentSnapshot
	targetField := fmt.Sprintf("_spatial.h3_map.r%d", opts.Resolution)

	batchSize := 30
	for i := 0; i < len(targetCells); i += batchSize {
		end := min(i+batchSize, len(targetCells))
		chunk := targetCells[i:end]

		query := c.fs.Collection(collection).Where(targetField, "in", chunk)
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}

		docs, err := query.Documents(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("executing firestore radius query: %w", err)
		}

		for _, doc := range docs {
			if !visitedDocs[doc.Ref.ID] {
				visitedDocs[doc.Ref.ID] = true
				allDocs = append(allDocs, doc)
			}
		}
	}

	// Post-filter documents via Haversine distance to discard false positives on bounding hexagon margins.
	var filteredDocs []*firestore.DocumentSnapshot
	for _, doc := range allDocs {
		data := doc.Data()
		spatialData, ok := data["_spatial"].(map[string]interface{})
		if !ok {
			continue
		}

		lat := parseCoordinate(spatialData["lat"])
		lng := parseCoordinate(spatialData["lng"])

		dist := haversineDistance(centerLat, centerLng, lat, lng)
		if dist <= radiusKm {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	return filteredDocs, nil
}

// parseCoordinate normalizes numeric types returned by Firestore into float64.
func parseCoordinate(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int64:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0.0
	}
}
