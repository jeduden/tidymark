package yamlutil_test

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/yamlutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var aliasTests = []struct {
	name    string
	input   string
	wantErr bool
}{
	{"clean YAML", "title: Hello\nauthor: World\n", false},
	{"anchor definition", "base: &base\n  name: foo\n", true},
	{"alias reference", "child:\n  <<: *base\n", true},
	{"ampersand in quoted string", "title: \"Q&A Session\"\n", false},
	{"ampersand in single quoted string", "title: 'Q&A'\n", false},
	{"asterisk in quoted string", "note: \"use *bold* text\"\n", false},
	{"ampersand in unquoted value", "title: Q&A\n", false},
	{"billion laughs chain", "a: &a [\"lol\"]\nb: &b [*a,*a]\nc: &c [*b,*b]\n", true},
	{"empty input", "", false},
	{"asterisk not followed by identifier", "note: 5 * 3 = 15\n", false},
	{"anchor at start of line", "&anchor value\n", true},
	{"alias at start of value", "key: *alias\n", true},
	{"block scalar with ampersand", "key: |\n  &name in block\n", false},
	{"block scalar with asterisk", "key: >\n  *name in folded\n", false},
	{"comment with ampersand", "key: val # &anchor\n", false},
	{"comment with asterisk", "key: val # *alias\n", false},
	{"escaped quote in double string", "key: \"she said \\\"&hello\\\"\"\n", false},
	{"doubled single quote", "key: 'it''s &here'\n", false},
	{"syntax error ignored", "key: [unclosed\n", false},
	{"multi-document clean", "title: a\n---\ntitle: b\n", false},
}

func TestRejectYAMLAliases(t *testing.T) {
	for _, tt := range aliasTests {
		t.Run(tt.name, func(t *testing.T) {
			err := yamlutil.RejectYAMLAliases([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "anchors/aliases are not permitted")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalSafe(t *testing.T) {
	t.Run("unmarshals clean YAML into struct", func(t *testing.T) {
		var out struct {
			Title string `yaml:"title"`
		}
		err := yamlutil.UnmarshalSafe([]byte("title: Hello\n"), &out)
		require.NoError(t, err)
		assert.Equal(t, "Hello", out.Title)
	})

	t.Run("rejects anchor/alias", func(t *testing.T) {
		var out any
		err := yamlutil.UnmarshalSafe([]byte("a: &a [\"lol\"]\nb: &b [*a,*a]\n"), &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anchors/aliases are not permitted")
	})

	t.Run("returns error on invalid YAML", func(t *testing.T) {
		var out struct{ A int }
		err := yamlutil.UnmarshalSafe([]byte("a: not-a-number\n"), &out)
		require.Error(t, err)
	})

	t.Run("empty input produces zero value", func(t *testing.T) {
		var out struct{ Title string }
		err := yamlutil.UnmarshalSafe([]byte(""), &out)
		require.NoError(t, err)
		assert.Equal(t, "", out.Title)
	})
}

func TestUnmarshalNodeSafe(t *testing.T) {
	t.Run("returns document node for clean YAML", func(t *testing.T) {
		node, err := yamlutil.UnmarshalNodeSafe([]byte("title: Hello\n"))
		require.NoError(t, err)
		assert.Equal(t, yaml.DocumentNode, node.Kind)
		require.NotEmpty(t, node.Content)
		assert.Equal(t, yaml.MappingNode, node.Content[0].Kind)
	})

	t.Run("rejects anchor/alias", func(t *testing.T) {
		_, err := yamlutil.UnmarshalNodeSafe([]byte("a: &a [\"lol\"]\nb: &b [*a,*a]\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anchors/aliases are not permitted")
	})

	t.Run("returns error on invalid YAML", func(t *testing.T) {
		_, err := yamlutil.UnmarshalNodeSafe([]byte("key: {unclosed\n"))
		require.Error(t, err)
	})

	t.Run("empty input returns empty node", func(t *testing.T) {
		node, err := yamlutil.UnmarshalNodeSafe([]byte(""))
		require.NoError(t, err)
		assert.Equal(t, yaml.Node{}, node)
	})
}

func TestMarshal(t *testing.T) {
	t.Run("marshals struct to YAML", func(t *testing.T) {
		v := struct {
			Title string `yaml:"title"`
		}{Title: "Hello"}
		data, err := yamlutil.Marshal(v)
		require.NoError(t, err)
		assert.Contains(t, string(data), "title: Hello")
	})

	t.Run("marshals nil to null YAML", func(t *testing.T) {
		data, err := yamlutil.Marshal(nil)
		require.NoError(t, err)
		assert.Equal(t, "null\n", string(data))
	})
}
