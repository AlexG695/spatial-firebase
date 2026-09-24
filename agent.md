# AGENT.md - `spatial-firebase` Architecture & Developer Guide

## 1. Overview & Purpose

`spatial-firebase` is a Go library that adds geospatial indexing and querying capabilities to **Google Cloud Firestore** using Uber's **H3 Hierarchical Hexagonal Spatial Index** (`github.com/uber/h3-go/v3`).

Because Cloud Firestore does not provide native geospatial indexing primitives (such as PostGIS or MongoDB 2dsphere), this library decomposes geographic coordinates into discrete H3 hexagonal cell indices. Spatial queries are transformed into indexed Firestore equality and `in` array operations, followed by an in-memory post-filter to eliminate false positives.

---

## 2. Repository Layout

```
.
├── client.go           # Spatial client wrapper and document write operations
├── spatial.go          # Core models (SpatialFeature, Coordinate) and geometry factories
├── query.go           # Radius search, H3 k-ring generation, query batching, Haversine filtering
├── spatial_test.go     # Unit test suite for spatial features and geometry generation
├── integration_test.go # End-to-end integration tests using the Firestore Emulator
├── go.mod              # Go module definition (Go 1.27)
├── go.sum              # Dependency checksums
├── README.md           # User documentation
└── agent.md            # Agent-oriented architectural blueprint and guide
```

---

## 3. Data Architecture & Storage Model

All spatial metadata is stored within Firestore documents under the reserved top-level field `_spatial`.

### Schema: `SpatialFeature`

```go
type SpatialFeature struct {
    Type     GeometryType      `firestore:"type" json:"type"`             // "Point" | "Polygon"
    Lat      float64           `firestore:"lat,omitempty" json:"lat"`     // Latitude (WGS84)
    Lng      float64           `firestore:"lng,omitempty" json:"lng"`     // Longitude (WGS84)
    H3Map    map[string]string `firestore:"h3_map,omitempty" json:"h3_map"` // Resolution mapping: {"r6": "hex", "r8": "hex", "r10": "hex"}
    H3Cells  []string          `firestore:"h3_cells,omitempty" json:"h3_cells"` // Polyfill cells (for Polygon)
    H3Center string            `firestore:"h3_center,omitempty" json:"h3_center"` // Centroid H3 index
}
```

### Geometry Types & Factories

1. **Point (`NewPoint(lat, lng float64, resolutions []int)`)**:
   - Computes H3 indices across multiple resolutions (e.g., `r6`, `r8`, `r10`).
   - Populates `H3Map` with `r<res> -> hexStr` pairs.
   - Automatically computes resolution 8 as the default `H3Center` centroid if omitted from `resolutions`.

2. **Polygon (`NewPolygon(coordinates []Coordinate, resolution int)`)**:
   - Requires at least 3 coordinates.
   - Calculates the polygon boundary outer loop.
   - Uses `h3.Polyfill` to generate all covering H3 cells at the given resolution, saved in `H3Cells`.
   - Computes the arithmetic centroid coordinate and its corresponding H3 index in `H3Center`.

### Document Writes

- Use `client.SetWithSpatial(ctx, docRef, data, feature)`.
- Injects the `_spatial` field into `data`.
- Persists with `firestore.MergeAll` to preserve non-spatial fields.

---

## 4. Query Architecture

Radius queries are executed via `client.SearchRadius(ctx, collection, centerLat, centerLng, radiusKm, opts)` through a two-phase query pipeline:

```
[Search Center (Lat, Lng) + Radius]
                 │
                 ▼
     [H3 Indexing & k-Ring Expansion]
   (Compute center hex + KRing neighbors)
                 │
                 ▼
      [Firestore Batch Queries]
   (Chunk cells into <= 30 items for 'in')
                 │
                 ▼
       [Deduplicate Documents]
    (Track visited document IDs)
                 │
                 ▼
    [In-Memory Haversine Distance]
  (Drop false positives outside radiusKm)
                 │
                 ▼
         [Result Snapshots]
```

### Pipeline Details

1. **Resolution Selection**: Defaults to `Resolution: 8` if unspecified.
2. **k-Ring Estimation**: Computes `kRing := int(math.Ceil(radiusKm / edgeLengthKm))` using the reference edge length table `h3EdgeLengthsKm`.
3. **Firestore Constraint Management**:
   - Firestore limits `in` queries to a maximum of **30 elements**.
   - Cells are batched into chunks of up to 30.
   - Each chunk executes a query: `.Where("_spatial.h3_map.r" + res, "in", chunk)`.
4. **Deduplication**: Results across batches are keyed by `doc.Ref.ID` in a visited map.
5. **Haversine Distance Post-Filtering**: Eliminates candidates residing in the outer hexagon margin that fall outside the true Euclidean/great-circle radius.
6. **Numeric Coercion**: `parseCoordinate` handles Firestore deserialization nuances where numbers may unmarshal as `float64`, `float32`, `int64`, or `int`.

---

## 5. Development & Testing

### Prerequisites
- Go 1.27+
- (Optional for integration tests) Firebase CLI with Firestore emulator installed.

### Commands

| Task | Command |
|---|---|
| Run unit tests | `go test -v ./...` |
| Run tests with race detector | `go test -race -v ./...` |
| Static analysis | `go vet ./...` |
| Format codebase | `gofmt -s -w .` |
| Run integration tests | `FIRESTORE_EMULATOR_HOST=localhost:8080 go test -v -run TestFirestoreEmulator_Integration ./...` |
| Start Firestore emulator | `firebase emulators:start --only firestore` |

---

## 6. Firestore Indexing Requirements

- **Single-Field Indexes**: Standard queries on `_spatial.h3_map.r<res>` use Firestore's default automatic single-field indexes.
- **Composite Indexes**: If chaining spatial queries with application filters (e.g., `status == "active"` AND `_spatial.h3_map.r8 in [...]`), configure a composite index in `firestore.indexes.json`:

```json
{
  "indexes": [
    {
      "collectionGroup": "vehicles",
      "queryScope": "COLLECTION",
      "fields": [
        { "fieldPath": "status", "order": "ASCENDING" },
        { "fieldPath": "_spatial.h3_map.r8", "order": "ASCENDING" }
      ]
    }
  ]
}
```

---

## 7. Guidelines for Agents Modifying this Codebase

1. **Context & Error Handling**:
   - Always propagate `context.Context` to Firestore operations.
   - Wrap errors with technical context using `%w` (e.g., `fmt.Errorf("executing firestore radius query: %w", err)`).
2. **Comment Conventions**:
   - Write concise, technical doc comments in English.
   - Adhere to Effective Go doc conventions (`// SymbolName does ...`).
   - Avoid redundant inline comments that restate self-evident Go code.
3. **Firestore Data Parsing**:
   - Always use or extend `parseCoordinate` when reading coordinates from document snapshots. Do not cast directly to `float64`.
4. **Resolution Tuning**:
   - Avoid using resolutions below 6 or above 10 without updating `h3EdgeLengthsKm`.
   - Keep in mind that high resolutions combined with large search radii produce exponential k-ring expansions, leading to excessive Firestore reads.
