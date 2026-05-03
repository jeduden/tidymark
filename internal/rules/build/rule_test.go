package build

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFile parses inline markdown into a lint.File.
func newFile(t *testing.T, src string) *lint.File {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	return f
}

// --- Metadata ---

func TestRule_ID(t *testing.T) {
	assert.Equal(t, "MDS039", (&Rule{}).ID())
}

func TestRule_Name(t *testing.T) {
	assert.Equal(t, "build", (&Rule{}).Name())
}

func TestRule_Category(t *testing.T) {
	assert.Equal(t, "meta", (&Rule{}).Category())
}

// --- DefaultSettings / ApplySettings ---

func TestDefaultSettings_Empty(t *testing.T) {
	assert.Empty(t, (&Rule{}).DefaultSettings())
}

func TestApplySettings_UnknownKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"bogus": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown setting")
}

func TestApplySettings_Recipes_Valid(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"recipes": map[string]any{
			"chart": map[string]any{
				"body-template": "![{alt}]({output})",
				"params": map[string]any{
					"required": []any{"data"},
					"optional": []any{"title"},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Contains(t, r.recipes, "chart")
	schema := r.recipes["chart"]
	assert.Equal(t, "![{alt}]({output})", schema.BodyTemplate)
	assert.Equal(t, []string{"data"}, schema.Required)
	assert.Equal(t, []string{"title"}, schema.Optional)
}

func TestApplySettings_Recipes_NotMap(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"recipes": "not-a-map"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipes must be a map")
}

// --- resolveRecipe ---

func TestResolveRecipe_Builtin_Screenshot(t *testing.T) {
	r := &Rule{}
	schema, ok := r.resolveRecipe("screenshot")
	require.True(t, ok)
	assert.Contains(t, schema.Required, "url")
}

func TestResolveRecipe_Builtin_VHS(t *testing.T) {
	r := &Rule{}
	schema, ok := r.resolveRecipe("vhs")
	require.True(t, ok)
	assert.Contains(t, schema.Required, "input")
}

func TestResolveRecipe_Unknown(t *testing.T) {
	r := &Rule{}
	_, ok := r.resolveRecipe("nonexistent")
	assert.False(t, ok)
}

func TestResolveRecipe_UserDeclared(t *testing.T) {
	r := &Rule{
		recipes: map[string]recipeSchema{
			"chart": {Required: []string{"data"}},
		},
	}
	schema, ok := r.resolveRecipe("chart")
	require.True(t, ok)
	assert.Equal(t, []string{"data"}, schema.Required)
}

// --- validateHard ---

func TestValidateHard_MissingRecipe(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{"output": "out.png"})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `missing required "recipe"`)
}

func TestValidateHard_MissingOutput(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{"recipe": "vhs", "input": "demo.tape"})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `missing required "output"`)
}

func TestValidateHard_DotDotOutput(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{
		"recipe": "vhs", "input": "demo.tape", "output": "../out/file.gif",
	})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `".." path component`)
}

func TestValidateHard_UnknownRecipe(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{
		"recipe": "nope", "output": "out.png",
	})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `unknown recipe "nope"`)
}

func TestValidateHard_MissingRequiredParam(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{
		"recipe": "screenshot", "output": "out.png",
	})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `missing required parameter "url"`)
}

func TestValidateHard_Valid_Screenshot(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{
		"recipe": "screenshot", "output": "out.png", "url": "/home",
	})
	assert.Empty(t, diags)
}

func TestValidateHard_Valid_VHS(t *testing.T) {
	r := &Rule{}
	diags := r.validateHard("test.md", 1, map[string]string{
		"recipe": "vhs", "output": "demo.gif", "input": "demo.tape",
	})
	assert.Empty(t, diags)
}

// --- warnUnknownParams ---

func TestWarnUnknownParams_Clean(t *testing.T) {
	r := &Rule{}
	schema := builtinRecipes["vhs"]
	diags := r.warnUnknownParams("test.md", 1, "vhs", schema, map[string]string{
		"recipe": "vhs", "output": "demo.gif", "input": "demo.tape",
	})
	assert.Empty(t, diags)
}

func TestWarnUnknownParams_Unknown(t *testing.T) {
	r := &Rule{}
	schema := builtinRecipes["vhs"]
	diags := r.warnUnknownParams("test.md", 1, "vhs", schema, map[string]string{
		"recipe": "vhs", "output": "demo.gif", "input": "demo.tape", "extra": "val",
	})
	require.Len(t, diags, 1)
	assert.Equal(t, lint.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Message, `unknown parameter "extra"`)
}

func TestWarnUnknownParams_Sorted(t *testing.T) {
	r := &Rule{}
	schema := builtinRecipes["vhs"]
	diags := r.warnUnknownParams("test.md", 1, "vhs", schema, map[string]string{
		"recipe": "vhs", "output": "demo.gif", "input": "demo.tape",
		"zzz": "1", "aaa": "2",
	})
	require.Len(t, diags, 2)
	assert.Contains(t, diags[0].Message, `"aaa"`)
	assert.Contains(t, diags[1].Message, `"zzz"`)
}

// --- generateBody ---

func TestGenerateBody_Screenshot_DefaultAlt(t *testing.T) {
	r := &Rule{}
	body, diags := r.generateBody("test.md", 1, map[string]string{
		"recipe": "screenshot", "output": "docs/out.png", "url": "/home",
	})
	require.Empty(t, diags)
	assert.Equal(t, "![screenshot output: docs/out.png](docs/out.png)\n", body)
}

func TestGenerateBody_VHS_DefaultAlt(t *testing.T) {
	r := &Rule{}
	body, diags := r.generateBody("test.md", 1, map[string]string{
		"recipe": "vhs", "output": "demo.gif", "input": "demo.tape",
	})
	require.Empty(t, diags)
	assert.Equal(t, "![vhs output: demo.gif](demo.gif)\n", body)
}

func TestGenerateBody_CustomRecipe_DefaultTemplate(t *testing.T) {
	r := &Rule{
		recipes: map[string]recipeSchema{
			"chart": {Required: []string{"data"}},
		},
	}
	body, diags := r.generateBody("test.md", 1, map[string]string{
		"recipe": "chart", "output": "chart.png", "data": "data.csv",
	})
	require.Empty(t, diags)
	assert.Equal(t, "[chart.png](chart.png)\n", body)
}

func TestGenerateBody_CustomRecipe_CustomTemplate(t *testing.T) {
	r := &Rule{
		recipes: map[string]recipeSchema{
			"chart": {
				Required:     []string{"data"},
				BodyTemplate: "![{alt}]({output})",
			},
		},
	}
	body, diags := r.generateBody("test.md", 1, map[string]string{
		"recipe": "chart", "output": "chart.png", "data": "data.csv",
	})
	require.Empty(t, diags)
	assert.Equal(t, "![chart output: chart.png](chart.png)\n", body)
}

// --- Check (integration) ---

func TestCheck_NoDirectives(t *testing.T) {
	r := &Rule{}
	f := newFile(t, "# Hello\n\nNo directives here.\n")
	assert.Empty(t, r.Check(f))
}

func TestCheck_CorrectBody_VHS(t *testing.T) {
	r := &Rule{}
	src := "# Demo\n\n<?build\nrecipe: vhs\ninput: demo.tape\noutput: demo.gif\n?>\n" +
		"![vhs output: demo.gif](demo.gif)\n<?/build?>\n"
	f := newFile(t, src)
	assert.Empty(t, r.Check(f))
}

func TestCheck_StaleBody(t *testing.T) {
	r := &Rule{}
	src := "# Demo\n\n<?build\nrecipe: vhs\ninput: demo.tape\noutput: demo.gif\n?>\nwrong content\n<?/build?>\n"
	f := newFile(t, src)
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "out of date")
}

func TestCheck_UnknownRecipe(t *testing.T) {
	r := &Rule{}
	src := "# Test\n\n<?build\nrecipe: ghost\noutput: out.png\n?>\ncontent\n<?/build?>\n"
	f := newFile(t, src)
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `unknown recipe "ghost"`)
}

func TestCheck_UnknownParam_AndCorrectBody(t *testing.T) {
	r := &Rule{}
	src := "# Demo\n\n<?build\nrecipe: vhs\ninput: demo.tape\noutput: demo.gif\nextra: val\n?>\n" +
		"![vhs output: demo.gif](demo.gif)\n<?/build?>\n"
	f := newFile(t, src)
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, lint.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Message, `unknown parameter "extra"`)
}

func TestCheck_UnknownParam_AndStaleBody(t *testing.T) {
	r := &Rule{}
	src := "# Demo\n\n<?build\nrecipe: vhs\ninput: demo.tape\noutput: demo.gif\nextra: val\n?>\nwrong\n<?/build?>\n"
	f := newFile(t, src)
	diags := r.Check(f)
	require.Len(t, diags, 2)
	// Warning for unknown param
	assert.Equal(t, lint.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Message, `unknown parameter "extra"`)
	// Error for stale body
	assert.Equal(t, lint.Error, diags[1].Severity)
	assert.Contains(t, diags[1].Message, "out of date")
}

// --- Fix ---

func TestFix_RegeneratesBody_VHS(t *testing.T) {
	r := &Rule{}
	src := "# Demo\n\n<?build\nrecipe: vhs\ninput: demo.tape\noutput: demo.gif\n?>\nwrong content\n<?/build?>\n"
	f := newFile(t, src)
	got := string(r.Fix(f))
	assert.Contains(t, got, "![vhs output: demo.gif](demo.gif)")
	assert.NotContains(t, got, "wrong content")
}

func TestFix_RegeneratesBody_Screenshot(t *testing.T) {
	r := &Rule{}
	src := "# Page\n\n<?build\nrecipe: screenshot\nurl: /home\noutput: docs/home.png\n?>\nstale\n<?/build?>\n"
	f := newFile(t, src)
	got := string(r.Fix(f))
	assert.Contains(t, got, "![screenshot output: docs/home.png](docs/home.png)")
}

func TestFix_SkipsInvalidDirective(t *testing.T) {
	r := &Rule{}
	src := "# Test\n\n<?build\nrecipe: ghost\noutput: out.png\n?>\ncontent\n<?/build?>\n"
	f := newFile(t, src)
	// Fix should not panic and should leave content unchanged
	got := r.Fix(f)
	assert.Equal(t, src, string(got))
}

// --- hasDotDotSegment ---

func TestHasDotDotSegment(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"../out.png", true},
		{"a/../b.png", true},
		{"a/b/c.png", false},
		{"out.png", false},
		{"./out.png", false},
		{"..", true},
		{"a/b/..c.png", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, hasDotDotSegment(c.path), "path=%q", c.path)
	}
}

// --- output extension filter ---

func TestValidateHard_AnyExtension(t *testing.T) {
	r := &Rule{}
	for _, ext := range []string{"out.gif", "out.mp4", "out.svg", "out.txt", "out"} {
		diags := r.validateHard("test.md", 1, map[string]string{
			"recipe": "vhs", "output": ext, "input": "demo.tape",
		})
		assert.Empty(t, diags, "extension %q should be accepted", ext)
	}
}
