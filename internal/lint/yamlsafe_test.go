package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRejectYAMLAliases_FrontMatter verifies that ParseFrontMatterKinds
// rejects YAML anchors/aliases via the yamlutil safe-unmarshal path.
func TestRejectYAMLAliases_FrontMatter(t *testing.T) {
	t.Run("anchor in front matter is rejected", func(t *testing.T) {
		doc := []byte("---\na: &a [\"lol\"]\nb: &b [*a,*a]\nc: &c [*b,*b]\nkinds: [doc]\n---\n# Title\n")
		prefix, content := StripFrontMatter(doc)
		require.NotNil(t, prefix)
		assert.Contains(t, string(content), "# Title")

		_, err := ParseFrontMatterKinds(prefix)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anchors/aliases are not permitted")
	})

	t.Run("clean front matter is accepted", func(t *testing.T) {
		doc := []byte("---\ntitle: \"Q&A Guide\"\nkinds: [doc]\n---\n# Title\n")
		prefix, _ := StripFrontMatter(doc)
		require.NotNil(t, prefix)

		kinds, err := ParseFrontMatterKinds(prefix)
		require.NoError(t, err)
		assert.Equal(t, []string{"doc"}, kinds)
	})
}
