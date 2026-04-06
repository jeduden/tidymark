package gensection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYAMLBody_RejectsAnchor(t *testing.T) {
	mp := MarkerPair{
		StartLine: 1,
		YAMLBody:  "base: &base\n  key: val\n",
	}
	_, diags := ParseYAMLBody("test.md", mp, "MDS999", "test-rule")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "anchors/aliases are not permitted")
}

func TestParseYAMLBody_AcceptsClean(t *testing.T) {
	mp := MarkerPair{
		StartLine: 1,
		YAMLBody:  "key: value\n",
	}
	raw, diags := ParseYAMLBody("test.md", mp, "MDS999", "test-rule")
	assert.Empty(t, diags)
	assert.Equal(t, "value", raw["key"])
}
