package spatial

import (
	"context"

	"cloud.google.com/go/firestore"
)

// Client wraps a Firestore client to provide spatial indexing and querying capabilities.
type Client struct {
	fs *firestore.Client
}

// NewClient instantiates a new spatial Client with the provided Firestore client.
func NewClient(fsClient *firestore.Client) *Client {
	return &Client{fs: fsClient}
}

// SetWithSpatial writes or updates a Firestore document, injecting spatial indexing metadata into the "_spatial" field with merge semantics.
func (c *Client) SetWithSpatial(ctx context.Context, docRef *firestore.DocumentRef, data map[string]interface{}, feature SpatialFeature) error {
	if data == nil {
		data = make(map[string]interface{})
	}

	data["_spatial"] = feature

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	return err
}
