package spatial_test

import (
	"context"
	"os"
	spatial "spatial-firebase"
	"testing"

	"cloud.google.com/go/firestore"
)

// TestFirestoreEmulator_Integration verifies end-to-end spatial writes and radius search against the Firestore emulator.
func TestFirestoreEmulator_Integration(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("skipping integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()

	fsClient, err := firestore.NewClient(ctx, "demo-project")
	if err != nil {
		t.Fatalf("connecting to firestore emulator: %v", err)
	}
	defer fsClient.Close()

	spatialClient := spatial.NewClient(fsClient)
	collectionName := "test_vehicles_" + t.Name()

	// Seed test document with multi-resolution spatial index (Zocalo, Mexico City).
	docRef := fsClient.Collection(collectionName).NewDoc()
	pointData := map[string]interface{}{
		"driver_id": "drv_123",
		"status":    "active",
	}

	zocaloPoint := spatial.NewPoint(19.4326, -99.1332, []int{6, 8, 10})
	err = spatialClient.SetWithSpatial(ctx, docRef, pointData, zocaloPoint)
	if err != nil {
		t.Fatalf("setting document with spatial metadata: %v", err)
	}

	// Radius search: 2.0 km from Angel of Independence (~3.2 km from target; expected 0 results).
	resultsOut, err := spatialClient.SearchRadius(ctx, collectionName, 19.4270, -99.1677, 2.0, spatial.SearchOptions{Resolution: 8})
	if err != nil {
		t.Fatalf("executing SearchRadius (2km): %v", err)
	}
	if len(resultsOut) != 0 {
		t.Errorf("expected 0 results for 2.0 km radius, got: %d", len(resultsOut))
	}

	// Radius search: 5.0 km from Angel of Independence (expected 1 result).
	resultsIn, err := spatialClient.SearchRadius(ctx, collectionName, 19.4270, -99.1677, 5.0, spatial.SearchOptions{Resolution: 8})
	if err != nil {
		t.Fatalf("executing SearchRadius (5km): %v", err)
	}
	if len(resultsIn) != 1 {
		t.Errorf("expected 1 result for 5.0 km radius, got: %d", len(resultsIn))
	} else {
		t.Logf("spatial query successful, matched doc ID: %s", resultsIn[0].Ref.ID)
	}
}
