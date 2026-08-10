// Uniform grid spatial index for Story EF-003 (Spatial Index).
//
// Architecture: ADR-0001 CR-6, TR-ef-003. World space is divided into cells
// of spatial_bucket_size x spatial_bucket_size units; each cell key
// [2]int32{floor(x / size), floor(y / size)} maps to the GameEntityIDs whose
// positions fall inside it. Range queries enumerate the buckets overlapping
// the query circle's bounding box and hand the candidates to the store, which
// filters them by exact distance and component requirements.
//
// The Kaiju Transform is the authoritative position source (ADR-0001): the
// index tracks only bucket membership, never the transform itself.
// CreateEntity inserts on creation, retire removes on destruction, and
// systems call UpdatePosition after moving an entity's transform. Suspension
// changes no position, so it never reindexes.
package entity

import (
	"math"

	"kaijuengine.com/matrix"
)

// spatialIndex is the uniform grid over the X/Y plane (Story EF-003).
type spatialIndex struct {
	// bucketSize is the cell edge length in world units
	// (GDD spatial_bucket_size, default 32).
	bucketSize float64
	// overflowThreshold is the bucket population above which a range query
	// degrades to a linear scan (Story EF-003 AC-5, default 500).
	overflowThreshold int
	// buckets maps cell keys to the entity IDs whose positions fall in that
	// cell. Each indexed entity appears in exactly one bucket.
	buckets map[[2]int32][]GameEntityID
	// bucketOf tracks each indexed entity's current cell so a position
	// update can migrate the entity without scanning bucket slices.
	bucketOf map[GameEntityID][2]int32
	// degradedQueries counts range queries that degraded to a linear scan
	// because a scanned bucket exceeded overflowThreshold (AC-5 evidence).
	degradedQueries uint64
}

// newSpatialIndex creates an empty grid with the given cell size and overflow
// threshold.
func newSpatialIndex(bucketSize float64, overflowThreshold int) *spatialIndex {
	return &spatialIndex{
		bucketSize:        bucketSize,
		overflowThreshold: overflowThreshold,
		buckets:           make(map[[2]int32][]GameEntityID),
		bucketOf:          make(map[GameEntityID][2]int32),
	}
}

// keyFor returns the cell key containing the given position. Floor division
// places negative coordinates in the correct cell (GDD CR-6); plain integer
// truncation would map -1..0 onto cell 0.
func (idx *spatialIndex) keyFor(pos matrix.Vec3) [2]int32 {
	return [2]int32{
		int32(math.Floor(float64(pos.X()) / idx.bucketSize)),
		int32(math.Floor(float64(pos.Y()) / idx.bucketSize)),
	}
}

// insert registers an entity at the given position (creation-time placement).
// Inserting an already-indexed entity refreshes its cell instead of
// duplicating it.
func (idx *spatialIndex) insert(id GameEntityID, pos matrix.Vec3) {
	if oldKey, ok := idx.bucketOf[id]; ok && oldKey == idx.keyFor(pos) {
		return
	}
	idx.update(id, pos)
}

// update migrates an entity to the cell containing the given position after
// a move. Entities not yet indexed are inserted; moves within the same cell
// are no-ops. The caller is responsible for validating the entity.
func (idx *spatialIndex) update(id GameEntityID, pos matrix.Vec3) {
	key := idx.keyFor(pos)
	if oldKey, ok := idx.bucketOf[id]; ok {
		if oldKey == key {
			return
		}
		idx.removeFromBucket(id, oldKey)
	}
	idx.bucketOf[id] = key
	idx.buckets[key] = append(idx.buckets[key], id)
}

// remove drops an entity from the grid (destruction teardown). Removing an
// entity that was never indexed, or twice, is a no-op.
func (idx *spatialIndex) remove(id GameEntityID) {
	key, ok := idx.bucketOf[id]
	if !ok {
		return
	}
	delete(idx.bucketOf, id)
	idx.removeFromBucket(id, key)
}

// removeFromBucket drops an ID from the bucket slice at the given key.
func (idx *spatialIndex) removeFromBucket(id GameEntityID, key [2]int32) {
	bucket := idx.buckets[key]
	for i, e := range bucket {
		if e == id {
			idx.buckets[key] = append(bucket[:i], bucket[i+1:]...)
			return
		}
	}
}

// candidates returns the entity IDs in every bucket overlapping the circle of
// the given radius around center. The returned set is a superset of the
// entities within range: corner buckets whose cells the circle does not
// actually reach are included, and exact distance filtering is the caller's
// responsibility.
//
// The second result reports whether any scanned bucket exceeded the overflow
// threshold. In that case the caller must degrade to a linear scan (Story
// EF-003 AC-5); candidate collection is abandoned early because the caller
// discards it.
func (idx *spatialIndex) candidates(center matrix.Vec3, radius float64) ([]GameEntityID, bool) {
	minX := float64(center.X()) - radius
	maxX := float64(center.X()) + radius
	minY := float64(center.Y()) - radius
	maxY := float64(center.Y()) + radius
	x0 := int32(math.Floor(minX / idx.bucketSize))
	x1 := int32(math.Floor(maxX / idx.bucketSize))
	y0 := int32(math.Floor(minY / idx.bucketSize))
	y1 := int32(math.Floor(maxY / idx.bucketSize))

	var result []GameEntityID
	for bx := x0; bx <= x1; bx++ {
		for by := y0; by <= y1; by++ {
			key := [2]int32{bx, by}
			if len(idx.buckets[key]) > idx.overflowThreshold {
				return nil, true
			}
			result = append(result, idx.buckets[key]...)
		}
	}
	return result, false
}

// recordDegradation counts a range query that degraded to a linear scan. The
// store logs the warning; the counter is the observable evidence for Story
// EF-003 AC-5.
func (idx *spatialIndex) recordDegradation() {
	idx.degradedQueries++
}
