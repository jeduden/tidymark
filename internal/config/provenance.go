package config

import "fmt"

// LayerKind names a class of merge layer.
type LayerKind string

const (
	// LayerDefault marks the top-level rules: block in the user config
	// (the merge of built-in defaults and the loaded config). It is the
	// initial value of every leaf before any kind or override applies.
	LayerDefault LayerKind = "default"
	// LayerKindBody names a kind body. The Name field carries the
	// kind name.
	LayerKindBody LayerKind = "kind"
	// LayerOverride names an entry in the overrides: list. The Index
	// field carries the entry's position in the config.
	LayerOverride LayerKind = "override"
)

// LayerRef identifies one merge layer in a provenance chain.
type LayerRef struct {
	Kind  LayerKind // default, kind, override
	Name  string    // kind name when Kind=="kind"
	Index int       // override index when Kind=="override"
}

// String renders the layer as the form used by `mdsmith kinds resolve`:
// "default", "kinds.<name>", or "overrides[<i>]".
func (r LayerRef) String() string {
	switch r.Kind {
	case LayerKindBody:
		return "kinds." + r.Name
	case LayerOverride:
		return fmt.Sprintf("overrides[%d]", r.Index)
	default:
		return string(LayerDefault)
	}
}

// LeafProvenance records the value contributed by one layer to one leaf
// setting.
type LeafProvenance struct {
	Layer LayerRef
	Value any
}

// RuleProvenance is the per-leaf provenance for one rule. The keys are
// the same as RuleCfg.Settings; the value is the ordered list of layers
// that touched that leaf, oldest first.
type RuleProvenance struct {
	Enabled  []LeafProvenance
	Settings map[string][]LeafProvenance
}

// ProvenanceMap maps rule name to RuleProvenance.
type ProvenanceMap map[string]RuleProvenance

// EffectiveWithProvenance returns the effective rule configuration for a
// file path together with per-leaf provenance. The provenance map records
// every layer that contributed a value to each leaf setting, in the order
// they were applied; the last entry is the winning layer.
func EffectiveWithProvenance(
	cfg *Config, filePath string, fmKinds []string,
) (map[string]RuleCfg, ProvenanceMap) {
	kinds := resolveEffectiveKinds(cfg, filePath, fmKinds)
	rules := make(map[string]RuleCfg, len(cfg.Rules))
	prov := make(ProvenanceMap, len(cfg.Rules))

	defaultLayer := LayerRef{Kind: LayerDefault}
	for k, v := range cfg.Rules {
		rules[k] = copyRuleCfg(v)
		prov[k] = seedProvenance(v, defaultLayer)
	}

	apply := func(name string, layer RuleCfg, ref LayerRef) {
		base, ok := rules[name]
		if !ok {
			rules[name] = copyRuleCfg(layer)
			prov[name] = seedProvenance(layer, ref)
			return
		}
		merged := deepMergeRuleCfg(name, base, layer, defaultMergeModes)
		rules[name] = merged
		recordProvenance(name, prov, layer, ref)
	}

	for _, kindName := range kinds {
		body, ok := cfg.Kinds[kindName]
		if !ok {
			continue
		}
		ref := LayerRef{Kind: LayerKindBody, Name: kindName}
		for k, v := range body.Rules {
			apply(k, v, ref)
		}
	}
	for i, o := range cfg.Overrides {
		if !matchesAny(o.Files, filePath) {
			continue
		}
		ref := LayerRef{Kind: LayerOverride, Index: i}
		for k, v := range o.Rules {
			apply(k, v, ref)
		}
	}
	return rules, prov
}

// seedProvenance produces a RuleProvenance with a single entry per leaf
// — the value supplied at the named layer.
func seedProvenance(rc RuleCfg, ref LayerRef) RuleProvenance {
	out := RuleProvenance{
		Enabled:  []LeafProvenance{{Layer: ref, Value: rc.Enabled}},
		Settings: make(map[string][]LeafProvenance, len(rc.Settings)),
	}
	for k, v := range rc.Settings {
		out.Settings[k] = []LeafProvenance{{Layer: ref, Value: copyAnyValue(v)}}
	}
	return out
}

// recordProvenance appends a layer's contribution to the provenance of
// rule name. Only keys touched by the layer are amended; sibling keys
// from earlier layers retain their existing chain entries.
func recordProvenance(
	name string, prov ProvenanceMap, layer RuleCfg, ref LayerRef,
) {
	rp, ok := prov[name]
	if !ok {
		rp = RuleProvenance{Settings: map[string][]LeafProvenance{}}
	}
	rp.Enabled = append(rp.Enabled,
		LeafProvenance{Layer: ref, Value: layer.Enabled})
	if rp.Settings == nil {
		rp.Settings = map[string][]LeafProvenance{}
	}
	for k, v := range layer.Settings {
		rp.Settings[k] = append(rp.Settings[k],
			LeafProvenance{Layer: ref, Value: copyAnyValue(v)})
	}
	prov[name] = rp
}

// WinningLayer returns the layer that produced the final value of a
// leaf, or LayerDefault if the chain is empty. For an append-mode list
// the winning layer is the last layer that contributed; the merged
// list itself spans the full chain.
func (p RuleProvenance) WinningLayer(key string) LayerRef {
	chain := p.Settings[key]
	if len(chain) == 0 {
		// Fall back to enabled-bit chain so callers always get a layer.
		if len(p.Enabled) > 0 {
			return p.Enabled[len(p.Enabled)-1].Layer
		}
		return LayerRef{Kind: LayerDefault}
	}
	return chain[len(chain)-1].Layer
}
