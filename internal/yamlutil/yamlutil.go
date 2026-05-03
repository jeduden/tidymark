// Package yamlutil provides safe YAML parsing and marshaling helpers.
//
// All user-supplied content (config files, front matter, directive parameters)
// must pass through [UnmarshalSafe] or [UnmarshalNodeSafe] rather than calling
// yaml.Unmarshal directly. These wrappers call [RejectYAMLAliases] first, which
// prevents billion-laughs denial-of-service attacks by refusing any YAML that
// contains anchors or aliases before the alias expansion happens.
//
// When to use each function:
//   - [UnmarshalSafe] — unmarshal user content into a Go struct or map.
//   - [UnmarshalNodeSafe] — unmarshal user content into a raw yaml.Node tree
//     (needed when inspecting YAML structure before decoding into typed values).
//   - [Marshal] — thin wrapper around yaml.Marshal for consistency; safe for
//     output marshaling where data originates from trusted Go values.
//
// See docs/security/2026-04-05-adversarial-markdown.md for threat model context.
package yamlutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// RejectYAMLAliases decodes YAML into a node tree and returns an error if any
// anchor or alias is found. Decoding into yaml.Node does not expand aliases,
// so this is safe even for billion-laughs payloads. Non-anchor syntax errors
// return nil (handled by the caller's yaml.Unmarshal). This check must be
// called before yaml.Unmarshal on user-supplied content.
func RejectYAMLAliases(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			// An undefined alias causes a parse error containing "unknown anchor".
			// Reject this as evidence of alias usage.
			if strings.Contains(err.Error(), "unknown anchor") {
				return fmt.Errorf("yaml anchors/aliases are not permitted")
			}
			// Other syntax errors are handled by the caller's yaml.Unmarshal.
			return nil
		}

		if hasYAMLAnchorOrAlias(&doc) {
			return fmt.Errorf("yaml anchors/aliases are not permitted")
		}
	}
}

// UnmarshalSafe rejects YAML anchors/aliases then unmarshals data into v.
// Use this for all user-supplied YAML content (config files, front matter,
// directive parameters).
func UnmarshalSafe(data []byte, v any) error {
	if err := RejectYAMLAliases(data); err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

// UnmarshalNodeSafe rejects YAML anchors/aliases then unmarshals data into a
// yaml.Node document node. Use this when raw node inspection is needed before
// typed decoding (e.g. checking top-level key presence or tag types).
func UnmarshalNodeSafe(data []byte) (yaml.Node, error) {
	if err := RejectYAMLAliases(data); err != nil {
		return yaml.Node{}, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return yaml.Node{}, err
	}
	return node, nil
}

// Marshal is a thin wrapper around yaml.Marshal for consistency with
// UnmarshalSafe. Safe for output marshaling where data originates from
// trusted Go values.
func Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func hasYAMLAnchorOrAlias(node *yaml.Node) bool {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if hasYAMLAnchorOrAlias(child) {
			return true
		}
	}
	return false
}
