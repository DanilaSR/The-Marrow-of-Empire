package combat

import (
	"fmt"
	"sort"

	"marrow-of-empire/entity"
	"marrow-of-empire/stats"
)

// rollInitiative computes the fixed turn order for the encounter (AC-2,
// ADR-0009): initiative = DEX_mod + WIS_mod, ties broken by higher raw DEX,
// then by higher entity ID. The result is rolled once at combat start and
// never recalculated during the encounter (AC-2: "Turn order is fixed for the
// encounter duration").
//
// The comparator is a total order (initiative desc, DEX desc, entity ID
// desc), so the sort is fully deterministic and does not rely on slice
// stability. Negative modifiers are supported (DEX 6 = -2): initiative can be
// negative or zero (AC-2 edge case).
//
// Every combatant must carry an entity.Stats component (StartCombat
// validates this before rolling); a missing component returns an error naming
// the combatant.
func rollInitiative(store entity.EntityStore, entries []CombatantEntry) ([]TurnEntry, error) {
	order := make([]TurnEntry, 0, len(entries))
	for _, e := range entries {
		s, ok := entity.GetComponent[entity.Stats](store, e.EntityID)
		if !ok {
			return nil, fmt.Errorf("combat: combatant %d has no Stats component; cannot roll initiative", e.EntityID)
		}
		order = append(order, TurnEntry{
			EntityID:   e.EntityID,
			Side:       e.Side,
			Initiative: stats.Modifier(s.Dexterity) + stats.Modifier(s.Wisdom),
			Dexterity:  s.Dexterity,
		})
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].Initiative != order[j].Initiative {
			return order[i].Initiative > order[j].Initiative
		}
		if order[i].Dexterity != order[j].Dexterity {
			return order[i].Dexterity > order[j].Dexterity
		}
		return order[i].EntityID > order[j].EntityID
	})
	return order, nil
}
