// StatsComponent as an Entity Framework component (Story STAT-005, ADR-0001,
// ADR-0006). Registration lives in this package to avoid an import cycle:
// stats imports entity (events.go uses entity.EventBus and entity.GameEntityID),
// so entity cannot import stats. entity.RegisterComponent registers the type
// with pod and its typed store factory at init time; the typed accessors and
// serialization pass then treat it like any other component.
//
// StatsComponent fields are all pod-primitive ([NumAttributes]int array, int),
// so no RegisterPodType calls are needed.
//
// Example usage:
//
//	c := stats.StatsComponent{Attributes: [stats.NumAttributes]int{14, 12, 14, 10, 10, 8}, Level: 1}
//	entity.SetComponent(store, id, c)
//	got, ok := entity.GetComponent[stats.StatsComponent](store, id)
package stats

import "marrow-of-empire/entity"

func init() {
	entity.RegisterComponent[StatsComponent]()
}
