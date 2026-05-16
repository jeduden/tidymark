package rule

import "github.com/jeduden/mdsmith/internal/lint"

// Rule is a single lint rule that checks a Markdown file.
type Rule interface {
	ID() string
	Name() string
	Category() string
	Check(f *lint.File) []lint.Diagnostic
}

// FixableRule is a Rule that can also auto-fix violations.
type FixableRule interface {
	Rule
	Fix(f *lint.File) []byte
}

// Configurable is implemented by rules that have user-tunable settings.
type Configurable interface {
	ApplySettings(settings map[string]any) error
	DefaultSettings() map[string]any
}

// Defaultable is implemented by rules that override the default enabled
// state in generated/runtime configs.
type Defaultable interface {
	EnabledByDefault() bool
}

// MergeMode describes how a list-typed rule setting combines across
// config layers (defaults, kinds, overrides).
type MergeMode int

const (
	// MergeReplace is the default: a later layer's list replaces the
	// earlier layer's list wholesale.
	MergeReplace MergeMode = iota
	// MergeAppend concatenates a later layer's list onto the earlier
	// layer's list, preserving layer-chain order.
	MergeAppend
)

// ListMerger is implemented by Configurable rules that opt one or more
// list-typed settings out of the default MergeReplace behavior. The
// merge function calls SettingMergeMode(key) at config-resolution time
// and treats unknown keys as MergeReplace.
type ListMerger interface {
	SettingMergeMode(key string) MergeMode
}

// SettingsTranslator is implemented by Configurable rules that
// rewrite one config layer's settings map before the deep-merge
// runs. The config merge layer calls TranslateLayerSettings on
// every layer that configures the rule, so merge logic stays free
// of rule-name special cases.
//
// required-structure implements this to collapse the user-facing
// `schema:` / `inline-schema:` keys into an append-mode
// `schema-sources` list, letting multiple kinds compose their
// schemas instead of overwriting (plan 156).
type SettingsTranslator interface {
	// TranslateLayerSettings returns the settings the merge layer
	// should use for one layer. Implementations must treat the
	// input as read-only and return a new map when they change
	// anything; returning the input unchanged signals "no
	// translation applies".
	TranslateLayerSettings(settings map[string]any) map[string]any
}

// ConfigTarget is implemented by rules that validate the project
// config file (.mdsmith.yml) rather than individual Markdown files.
// The engine runner runs these rules once against a synthetic lint.File
// for the config file before per-file markdown processing; they return
// nil for all other file paths when configured in production mode.
type ConfigTarget interface {
	IsConfigFileRule() bool
}
