// .kg YAML parsing for the NPC Knowledge Graph (Story KG-002, TR-kg-004,
// ADR-0005). YAML .kg files are the sole authoring format for NPC knowledge —
// content is never embedded in Go code (control manifest Core Layer, forbidden
// approach). Files are parsed at scene load from
// assets/npc_knowledge/{region_id}/{npc_id}.kg.
//
// Every parse error carries the line of the offending content so designers can
// fix authoring mistakes without a dedicated tool (ADR-0005: "errors surface
// at load"). Invalid or unreadable files return an error; the scene loader
// then attaches FallbackPersonality with zero domains and zero facts (GDD edge
// case, story AC-2).
//
// Entity references in facts are numeric GameEntityIDs (uint64, ADR-0001) —
// the ADR-0005 authoring example shows symbolic names for illustration only.
package knowledge

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"marrow-of-empire/entity"
)

// FallbackPersonality is the personality attached to an NPC whose .kg file
// failed to load or parse (GDD "Edge Cases"): the NPC has zero domains and
// zero facts and can only say they don't know anything. The scene loader
// applies it after ParseKGFile returns an error.
const FallbackPersonality = "Confused, doesn't know what's happening."

// topFieldNames are the known top-level keys of a .kg document (ADR-0005).
// Unknown keys are rejected so authoring typos (e.g. "speach_style") surface
// at load time instead of being silently dropped.
var topFieldNames = map[string]bool{
	"npc_id":             true,
	"npc_name":           true,
	"domains":            true,
	"personality":        true,
	"speech_style":       true,
	"knows_not":          true,
	"knows_not_entities": true,
	"facts":              true,
}

// requiredTopFields must be present in every .kg document. knows_not and
// knows_not_entities are optional boundaries that default to empty.
var requiredTopFields = []string{
	"npc_id", "npc_name", "domains", "personality", "speech_style", "facts",
}

// factFieldNames are the known keys of a fact entry (ADR-0005 CR-2).
var factFieldNames = map[string]bool{
	"fact_id":            true,
	"domain":             true,
	"type":               true,
	"content":            true,
	"trust_required":     true,
	"references":         true,
	"discovery_triggers": true,
	"contradicts":        true,
	"leads_to":           true,
}

// ParseKG parses .kg YAML data (ADR-0005) into a KnowledgeGraph. It is the
// I/O-free core of ParseKGFile: all validation happens here so unit tests can
// exercise it with inline YAML.
//
// Validation (all errors carry a line number):
//   - empty data or an empty document is an error
//   - required fields npc_id, npc_name, domains, personality, speech_style,
//     and facts must be present; npc_id and npc_name must be non-empty
//   - unknown fields are rejected (typo detection)
//   - facts must have a unique, non-empty fact_id plus domain, type, and
//     content; type must be one of the four FactTypes; trust_required must be
//     in 0-3; each fact's domain must appear in the document's domains list
//
// DiscoveryTriggers are stored verbatim and never evaluated at parse time —
// trigger evaluation happens in QueryFacts (Story KG-003). On error the
// returned graph is zero-valued, so a caller can attach FallbackPersonality.
func ParseKG(data []byte) (graph KnowledgeGraph, err error) {
	defer func() {
		if r := recover(); r != nil {
			// yaml.v3 normally returns errors for malformed input, but
			// pathological documents (e.g. extreme nesting) could panic.
			// Per story AC-2, invalid input is an error, never a panic.
			graph = KnowledgeGraph{}
			err = fmt.Errorf("kg parse: panic while parsing yaml (converted to error): %v", r)
		}
	}()

	if len(bytes.TrimSpace(data)) == 0 {
		return KnowledgeGraph{}, errors.New("kg parse: empty file")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return KnowledgeGraph{}, fmt.Errorf("kg parse: invalid yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		return KnowledgeGraph{}, errors.New("kg parse: empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return KnowledgeGraph{}, fmt.Errorf("kg parse: document must be a mapping (line %d)", root.Line)
	}

	if err := validateRoot(root); err != nil {
		return KnowledgeGraph{}, err
	}
	if err := validateFacts(root); err != nil {
		return KnowledgeGraph{}, err
	}

	var kg KnowledgeGraph
	if err := root.Decode(&kg); err != nil {
		return KnowledgeGraph{}, fmt.Errorf("kg parse: %w", err)
	}
	return kg, nil
}

// ParseKGFile reads and parses a .kg YAML file (Story KG-002, AC-1). The
// caller resolves the ADR-0005 path convention assets/npc_knowledge/
// {region_id}/{npc_id}.kg; the parser does not enforce the location. On error
// the returned graph is zero-valued — the scene loader attaches
// FallbackPersonality with zero domains and zero facts (AC-2).
func ParseKGFile(path string) (KnowledgeGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KnowledgeGraph{}, fmt.Errorf("kg parse: read %s: %w", path, err)
	}
	graph, err := ParseKG(data)
	if err != nil {
		return KnowledgeGraph{}, fmt.Errorf("kg parse: %s: %w", path, err)
	}
	return graph, nil
}

// ValidateReferences checks every entity reference in graph against the
// entities present in the current region (Story KG-002, AC-3): Fact.References
// and Lead.EntityID. Unresolved references are reported through warn — the
// fact still loads (ADR-0005 validation criteria). entityExists must report
// whether a GameEntityID exists in the region being loaded; the scene loader
// passes the region's entity store, keeping this function free of engine
// dependencies. Returns the number of unresolved references. The graph is
// never modified.
func ValidateReferences(graph *KnowledgeGraph, entityExists func(entity.GameEntityID) bool, warn func(format string, args ...any)) int {
	if graph == nil || entityExists == nil {
		return 0
	}
	unresolved := 0
	for _, f := range graph.Facts {
		for _, id := range f.References {
			if !entityExists(id) {
				unresolved++
				if warn != nil {
					warn("kg: fact %q references unknown entity %d", f.FactID, id)
				}
			}
		}
		for _, lead := range f.LeadsTo {
			if !entityExists(lead.EntityID) {
				unresolved++
				if warn != nil {
					warn("kg: fact %q leads to unknown entity %d", f.FactID, lead.EntityID)
				}
			}
		}
	}
	return unresolved
}

// validateRoot checks the document's top-level mapping: unknown fields,
// required fields, and non-empty identity anchors.
func validateRoot(root *yaml.Node) error {
	if err := checkUnknownFields(root, topFieldNames, "in document"); err != nil {
		return err
	}
	for _, name := range requiredTopFields {
		if mappingField(root, name) == nil {
			return fmt.Errorf("kg parse: missing required field %q (near line %d)", name, root.Line)
		}
	}
	for _, name := range []string{"npc_id", "npc_name"} {
		if n := mappingField(root, name); n != nil {
			if v, ok := scalarValue(n); ok && v == "" {
				return fmt.Errorf("kg parse: field %q must not be empty (line %d)", name, n.Line)
			}
		}
	}
	return nil
}

// validateFacts checks the facts list: list shape, unknown fact fields,
// required fact fields, unique non-empty fact_ids, valid fact types, trust
// range, and domain membership.
func validateFacts(root *yaml.Node) error {
	facts := mappingField(root, "facts")
	if facts == nil {
		return nil // unreachable: validateRoot requires "facts"
	}
	if facts.Kind == yaml.ScalarNode {
		return nil // null is a valid empty facts list; bad scalars error in Decode
	}
	if facts.Kind != yaml.SequenceNode {
		return fmt.Errorf("kg parse: field %q must be a list (line %d)", "facts", facts.Line)
	}

	// Decode domains first so each fact's domain can be checked for
	// membership (Fact.Domain must be one of the graph's Domains, types.go).
	var domains []string
	domainsOK := false
	if dn := mappingField(root, "domains"); dn != nil {
		domainsOK = dn.Decode(&domains) == nil
	}
	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[d] = true
	}

	seen := make(map[string]int) // fact_id -> line of first occurrence
	for _, f := range facts.Content {
		if f.Kind != yaml.MappingNode {
			return fmt.Errorf("kg parse: each fact must be a mapping (line %d)", f.Line)
		}
		if err := checkUnknownFields(f, factFieldNames, "in fact"); err != nil {
			return err
		}

		idNode := mappingField(f, "fact_id")
		if idNode == nil {
			return fmt.Errorf("kg parse: missing required field %q in fact (near line %d)", "fact_id", f.Line)
		}
		id, ok := scalarValue(idNode)
		if !ok || id == "" {
			return fmt.Errorf("kg parse: field %q in fact must be a non-empty string (line %d)", "fact_id", idNode.Line)
		}
		if firstLine, dup := seen[id]; dup {
			return fmt.Errorf("kg parse: duplicate fact_id %q (line %d, first at line %d)", id, f.Line, firstLine)
		}
		seen[id] = f.Line

		for _, name := range []string{"domain", "type", "content"} {
			if mappingField(f, name) == nil {
				return fmt.Errorf("kg parse: missing required field %q in fact %q (near line %d)", name, id, f.Line)
			}
		}

		if tn := mappingField(f, "type"); tn != nil {
			t, _ := scalarValue(tn)
			switch FactType(t) {
			case FactType_Fact, FactType_Rumor, FactType_Opinion, FactType_Secret:
			default:
				return fmt.Errorf("kg parse: fact %q: unknown type %q (line %d); want FACT, RUMOR, OPINION, or SECRET", id, t, tn.Line)
			}
		}

		if tr := mappingField(f, "trust_required"); tr != nil {
			var v int
			if tr.Decode(&v) == nil && (v < 0 || v > 3) {
				return fmt.Errorf("kg parse: fact %q: trust_required %d out of range 0-3 (line %d)", id, v, tr.Line)
			}
		}

		if domainsOK {
			if dn := mappingField(f, "domain"); dn != nil {
				var d string
				if dn.Decode(&d) == nil && !domainSet[d] {
					return fmt.Errorf("kg parse: fact %q: domain %q not in domains list (line %d)", id, d, dn.Line)
				}
			}
		}
	}
	return nil
}

// checkUnknownFields rejects mapping keys not in the known set, so typos in
// field names surface at load time with a line number — the mitigation for
// ADR-0005's "no authoring-time validation" negative consequence.
func checkUnknownFields(m *yaml.Node, known map[string]bool, where string) error {
	for i := 0; i < len(m.Content); i += 2 {
		key := m.Content[i]
		if !known[key.Value] {
			return fmt.Errorf("kg parse: unknown field %q %s (line %d)", key.Value, where, key.Line)
		}
	}
	return nil
}

// mappingField returns the value node of the named key in a mapping, or nil.
func mappingField(m *yaml.Node, name string) *yaml.Node {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return m.Content[i+1]
		}
	}
	return nil
}

// scalarValue returns a scalar node's raw string value.
func scalarValue(n *yaml.Node) (string, bool) {
	if n == nil || n.Kind != yaml.ScalarNode {
		return "", false
	}
	return n.Value, true
}
