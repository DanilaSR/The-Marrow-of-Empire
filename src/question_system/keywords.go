// Default keyword sets for the nine standard knowledge domains (GDD CR-2).
// The matcher itself contains no domain vocabulary — keyword sets and the
// relevance threshold are data-driven (coding standards: gameplay values
// come from external config). These defaults are the GDD table; per-NPC
// custom keywords are authored in the NPC's .kg file (ADR-0005).
package questions

// DefaultKeywords are the standard keyword sets per domain, taken verbatim
// from the GDD CR-2 table. Matching is case-insensitive and whole-word;
// entries may be multi-word phrases ("how long"). "Ley" is capitalized in
// the GDD table and matches "ley" case-insensitively.
var DefaultKeywords = map[string][]string{
	"craft_metallurgy":    {"ore", "metal", "forge", "smith", "steel", "iron", "alloy", "weapon", "armor", "blade", "bronze", "copper", "silver"},
	"herbalism_medicine":  {"herb", "plant", "poison", "remedy", "heal", "wound", "sickness", "poultice", "root", "fungus"},
	"town_gossip":         {"rumor", "heard", "people", "town", "happen", "strange", "recently", "anyone", "visitor", "new"},
	"military_security":   {"guard", "patrol", "attack", "defend", "danger", "threat", "safe", "wall", "gate", "weapon", "soldier"},
	"old_empire_lore":     {"empire", "ruin", "ancient", "old", "Ley", "imperial", "history", "past", "fallen", "technology", "machine"},
	"wilderness_survival": {"forest", "river", "path", "animal", "beast", "track", "weather", "hunt", "forage", "camp"},
	"trade_commerce":      {"buy", "sell", "price", "gold", "trade", "merchant", "goods", "supply", "rare", "valuable"},
	"faction_politics":    {"faction", "power", "leader", "alliance", "enemy", "rival", "politics", "control", "influence"},
	"personal_history":    {"you", "your", "past", "before", "born", "family", "childhood", "name", "why", "how long"},
}

// DefaultDomain returns the domain with DefaultKeywords' keyword set for
// name. An unknown name yields a domain with no keywords (custom domains
// are authored per NPC in .kg files, ADR-0005).
func DefaultDomain(name string) Domain {
	return Domain{Name: name, Keywords: DefaultKeywords[name]}
}

// DefaultDomains maps each name to its default-keyword domain, preserving
// order. Convenience for building an NPC's domain list from
// KnowledgeGraph.Domains (ADR-0005).
func DefaultDomains(names ...string) []Domain {
	domains := make([]Domain, 0, len(names))
	for _, n := range names {
		domains = append(domains, DefaultDomain(n))
	}
	return domains
}
