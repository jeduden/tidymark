package catalog

import (
	"github.com/jeduden/tidymark/internal/archetype/gensection"
	"github.com/jeduden/tidymark/internal/lint"
)

// makeDiag creates a TM019 error diagnostic at the given line.
func makeDiag(filePath string, line int, message string) lint.Diagnostic {
	return gensection.MakeDiag("TM019", "catalog", filePath, line, message)
}

// parseColumnConfig converts the raw YAML columns map into the
// catalog-internal columnConfig type by delegating to gensection.
func parseColumnConfig(raw map[string]any) map[string]columnConfig {
	gc := gensection.ParseColumnConfig(raw)
	if gc == nil {
		return nil
	}
	result := make(map[string]columnConfig, len(gc))
	for k, v := range gc {
		result[k] = columnConfig{
			maxWidth: v.MaxWidth,
			wrap:     v.Wrap,
		}
	}
	return result
}

// fromGensectionColumns converts gensection ColumnConfig to catalog columnConfig.
func fromGensectionColumns(cols map[string]gensection.ColumnConfig) map[string]columnConfig {
	if cols == nil {
		return nil
	}
	result := make(map[string]columnConfig, len(cols))
	for k, v := range cols {
		result[k] = columnConfig{
			maxWidth: v.MaxWidth,
			wrap:     v.Wrap,
		}
	}
	return result
}
