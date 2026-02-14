package frontmattervalidation

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"gopkg.in/yaml.v3"
)

func init() {
	rule.Register(&Rule{
		Required: []string{},
		Fields:   map[string]FieldSchema{},
	})
}

// FieldSchema defines validation requirements for a single front matter field.
type FieldSchema struct {
	Type     string
	Enum     []any
	Items    *FieldSchema
	MinItems *int
	MaxItems *int
}

// Rule validates YAML front matter fields against a configured schema.
type Rule struct {
	Required []string
	Fields   map[string]FieldSchema
}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS027" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "front-matter-validation" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "meta" }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	if len(r.Required) == 0 && len(r.Fields) == 0 {
		return nil
	}

	fmBlock := frontMatterBlock(f)
	if len(fmBlock) == 0 {
		return r.requiredDiagnostics(f.Path, nil)
	}

	raw, err := parseFrontMatter(fmBlock)
	if err != nil {
		return []lint.Diagnostic{r.diag(
			f.Path,
			fmt.Sprintf("front matter is not valid YAML: %v", err),
		)}
	}

	diags := r.requiredDiagnostics(f.Path, raw)
	diags = append(diags, r.fieldDiagnostics(f.Path, raw)...)
	return diags
}

// ApplySettings implements rule.Configurable.
func (r *Rule) ApplySettings(settings map[string]any) error {
	for key, raw := range settings {
		switch key {
		case "required":
			required, err := parseRequired(raw)
			if err != nil {
				return fmt.Errorf(
					"front-matter-validation: %w", err,
				)
			}
			r.Required = required
		case "fields":
			fields, err := parseFields(raw)
			if err != nil {
				return fmt.Errorf(
					"front-matter-validation: %w", err,
				)
			}
			r.Fields = fields
		default:
			return fmt.Errorf(
				"front-matter-validation: unknown setting %q", key,
			)
		}
	}

	return nil
}

// DefaultSettings implements rule.Configurable.
func (r *Rule) DefaultSettings() map[string]any {
	return map[string]any{
		"required": []string{},
		"fields":   map[string]any{},
	}
}

func parseRequired(raw any) ([]string, error) {
	values, ok := toAnySlice(raw)
	if !ok {
		return nil, fmt.Errorf(
			`required must be a list of field names, got %T`,
			raw,
		)
	}

	required := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		field, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf(
				`required field names must be strings, got %T`,
				value,
			)
		}
		if seen[field] {
			continue
		}
		seen[field] = true
		required = append(required, field)
	}
	return required, nil
}

func parseFields(raw any) (map[string]FieldSchema, error) {
	fieldMap, ok := asStringAnyMap(raw)
	if !ok {
		return nil, fmt.Errorf(
			`fields must be a map of field name to schema, got %T`,
			raw,
		)
	}

	parsed := make(map[string]FieldSchema, len(fieldMap))
	for field, rawSchema := range fieldMap {
		schema, err := parseFieldSchema(field, rawSchema)
		if err != nil {
			return nil, err
		}
		parsed[field] = schema
	}

	return parsed, nil
}

func parseFieldSchema(field string, raw any) (FieldSchema, error) {
	if typeName, ok := raw.(string); ok {
		return parseFieldSchemaType(field, typeName)
	}

	cfg, ok := asStringAnyMap(raw)
	if !ok {
		return FieldSchema{}, fmt.Errorf(
			`fields.%s must be a string or mapping, got %T`,
			field, raw,
		)
	}

	return parseFieldSchemaMap(field, cfg)
}

func parseFieldSchemaType(
	field string, typeName string,
) (FieldSchema, error) {
	if err := validateTypeName(field, typeName); err != nil {
		return FieldSchema{}, err
	}
	return FieldSchema{Type: typeName}, nil
}

func parseFieldSchemaMap(
	field string, cfg map[string]any,
) (FieldSchema, error) {
	var result FieldSchema
	for key, value := range cfg {
		switch key {
		case "type":
			typeName, err := parseTypeSetting(field, value)
			if err != nil {
				return FieldSchema{}, err
			}
			result.Type = typeName
		case "enum":
			enumValues, err := parseEnumSetting(field, value)
			if err != nil {
				return FieldSchema{}, err
			}
			result.Enum = enumValues
		case "items":
			itemSchema, err := parseItemsSetting(field, value)
			if err != nil {
				return FieldSchema{}, err
			}
			result.Items = itemSchema
		case "min-items":
			n, err := parseArrayBoundSetting(
				field, "min-items", value,
			)
			if err != nil {
				return FieldSchema{}, err
			}
			result.MinItems = n
		case "max-items":
			n, err := parseArrayBoundSetting(
				field, "max-items", value,
			)
			if err != nil {
				return FieldSchema{}, err
			}
			result.MaxItems = n
		default:
			return FieldSchema{}, fmt.Errorf(
				`fields.%s has unknown setting %q`,
				field, key,
			)
		}
	}

	if err := validateFieldSchema(field, result); err != nil {
		return FieldSchema{}, err
	}
	return result, nil
}

func parseTypeSetting(field string, value any) (string, error) {
	typeName, ok := value.(string)
	if !ok {
		return "", fmt.Errorf(
			`fields.%s.type must be a string, got %T`,
			field, value,
		)
	}
	if err := validateTypeName(field, typeName); err != nil {
		return "", err
	}
	return typeName, nil
}

func validateTypeName(field string, typeName string) error {
	if supportedType(typeName) {
		return nil
	}
	return fmt.Errorf(
		`fields.%s.type must be one of %q, got %q`,
		field, supportedTypes(), typeName,
	)
}

func parseEnumSetting(field string, value any) ([]any, error) {
	enumValues, ok := toAnySlice(value)
	if !ok {
		return nil, fmt.Errorf(
			`fields.%s.enum must be a list, got %T`,
			field, value,
		)
	}
	return enumValues, nil
}

func parseItemsSetting(
	field string, value any,
) (*FieldSchema, error) {
	itemSchema, err := parseFieldSchema(field+".items", value)
	if err != nil {
		return nil, err
	}
	return &itemSchema, nil
}

func parseArrayBoundSetting(
	field, name string, value any,
) (*int, error) {
	n, ok := toInt(value)
	if !ok {
		return nil, fmt.Errorf(
			`fields.%s.%s must be an integer, got %T`,
			field, name, value,
		)
	}
	if n < 0 {
		return nil, fmt.Errorf(
			`fields.%s.%s must be >= 0, got %d`,
			field, name, n,
		)
	}
	return &n, nil
}

func validateFieldSchema(
	field string, schema FieldSchema,
) error {
	if schema.Type == "" &&
		len(schema.Enum) == 0 &&
		schema.Items == nil &&
		schema.MinItems == nil &&
		schema.MaxItems == nil {
		return fmt.Errorf(
			`fields.%s must set at least one validation setting`,
			field,
		)
	}

	if hasArraySettings(schema) && schema.Type != "array" {
		return fmt.Errorf(
			`fields.%s uses array settings but type is %q (must be "array")`,
			field, schema.Type,
		)
	}

	if schema.MinItems != nil && schema.MaxItems != nil {
		if *schema.MinItems > *schema.MaxItems {
			return fmt.Errorf(
				`fields.%s.min-items (%d) must be <= max-items (%d)`,
				field, *schema.MinItems, *schema.MaxItems,
			)
		}
	}

	if schema.Type == "" {
		return nil
	}

	for _, enumVal := range schema.Enum {
		msg := validationMessageForValue(field, enumVal, schema)
		if msg == "" {
			continue
		}
		return fmt.Errorf(
			`fields.%s.enum value %s is invalid: %s`,
			field, formatValue(enumVal), msg,
		)
	}
	return nil
}

func hasArraySettings(schema FieldSchema) bool {
	return schema.Items != nil ||
		schema.MinItems != nil ||
		schema.MaxItems != nil
}

func frontMatterBlock(f *lint.File) []byte {
	if len(f.FrontMatter) > 0 {
		return f.FrontMatter
	}
	prefix, _ := lint.StripFrontMatter(f.Source)
	return prefix
}

func parseFrontMatter(block []byte) (map[string]any, error) {
	yamlBytes, err := extractFrontMatterYAML(block)
	if err != nil {
		return nil, err
	}

	if len(bytes.TrimSpace(yamlBytes)) == 0 {
		return map[string]any{}, nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw, nil
}

func extractFrontMatterYAML(block []byte) ([]byte, error) {
	delim := []byte("---\n")
	if !bytes.HasPrefix(block, delim) || !bytes.HasSuffix(block, delim) {
		return nil, fmt.Errorf("malformed front matter delimiters")
	}
	return block[len(delim) : len(block)-len(delim)], nil
}

func (r *Rule) requiredDiagnostics(
	path string, raw map[string]any,
) []lint.Diagnostic {
	var diags []lint.Diagnostic
	for _, field := range r.Required {
		if _, ok := raw[field]; ok {
			continue
		}
		diags = append(diags, r.diag(path, fmt.Sprintf(
			`front matter missing required field %q`, field,
		)))
	}
	return diags
}

func (r *Rule) fieldDiagnostics(
	path string, raw map[string]any,
) []lint.Diagnostic {
	var fields []string
	for field := range r.Fields {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	var diags []lint.Diagnostic
	for _, field := range fields {
		schema := r.Fields[field]
		value, ok := raw[field]
		if !ok {
			continue
		}

		if msg := validationMessageForValue(field, value, schema); msg != "" {
			diags = append(diags, r.diag(path, msg))
		}
	}

	return diags
}

func validationMessageForValue(
	field string, value any, schema FieldSchema,
) string {
	if schema.Type != "" && !matchesType(value, schema.Type) {
		return fmt.Sprintf(
			`front matter field %q must be %s, got %s`,
			field, schema.Type, valueTypeName(value),
		)
	}

	if len(schema.Enum) > 0 && !valueInEnum(value, schema.Enum) {
		return fmt.Sprintf(
			`front matter field %q has invalid value %s (allowed: %s)`,
			field,
			formatValue(value),
			formatAllowedValues(schema.Enum),
		)
	}

	if schema.Type != "array" {
		return ""
	}

	values, ok := toAnySlice(value)
	if !ok {
		return ""
	}

	if schema.MinItems != nil && len(values) < *schema.MinItems {
		return fmt.Sprintf(
			`front matter field %q must have at least %d items, got %d`,
			field, *schema.MinItems, len(values),
		)
	}

	if schema.MaxItems != nil && len(values) > *schema.MaxItems {
		return fmt.Sprintf(
			`front matter field %q must have at most %d items, got %d`,
			field, *schema.MaxItems, len(values),
		)
	}

	if schema.Items == nil {
		return ""
	}

	for i, item := range values {
		itemField := fmt.Sprintf("%s[%d]", field, i)
		if msg := validationMessageForValue(
			itemField, item, *schema.Items,
		); msg != "" {
			return msg
		}
	}

	return ""
}

func (r *Rule) diag(path, message string) lint.Diagnostic {
	return lint.Diagnostic{
		File:     path,
		Line:     1,
		Column:   1,
		RuleID:   r.ID(),
		RuleName: r.Name(),
		Severity: lint.Error,
		Message:  message,
	}
}

func supportedTypes() []string {
	return []string{
		"string",
		"int",
		"number",
		"bool",
		"array",
		"object",
	}
}

func supportedType(typeName string) bool {
	return slices.Contains(supportedTypes(), typeName)
}

func matchesType(value any, typeName string) bool {
	switch typeName {
	case "string":
		_, ok := value.(string)
		return ok
	case "int":
		return isInt(value)
	case "number":
		return isNumber(value)
	case "bool":
		_, ok := value.(bool)
		return ok
	case "array":
		return value != nil && reflect.TypeOf(value).Kind() == reflect.Slice
	case "object":
		return value != nil && reflect.TypeOf(value).Kind() == reflect.Map
	default:
		return false
	}
}

func valueTypeName(value any) string {
	if value == nil {
		return "null"
	}
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	}
	if isInt(value) {
		return "int"
	}
	if isNumber(value) {
		return "number"
	}
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return kind.String()
	}
}

func valueInEnum(value any, allowed []any) bool {
	for _, candidate := range allowed {
		if equalValue(value, candidate) {
			return true
		}
	}
	return false
}

func equalValue(a, b any) bool {
	if isNumber(a) && isNumber(b) {
		af, okA := toFloat64(a)
		bf, okB := toFloat64(b)
		return okA && okB && af == bf
	}
	return reflect.DeepEqual(a, b)
}

func formatAllowedValues(values []any) string {
	formatted := make([]string, 0, len(values))
	for _, value := range values {
		formatted = append(formatted, formatValue(value))
	}
	return strings.Join(formatted, ", ")
}

func formatValue(value any) string {
	if value == nil {
		return "null"
	}
	if s, ok := value.(string); ok {
		return strconv.Quote(s)
	}
	return fmt.Sprintf("%v", value)
}

func asStringAnyMap(value any) (map[string]any, bool) {
	switch m := value.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for key, val := range m {
			name, ok := key.(string)
			if !ok {
				return nil, false
			}
			out[name] = val
		}
		return out, true
	default:
		return nil, false
	}
}

func toAnySlice(value any) ([]any, bool) {
	if value == nil {
		return nil, false
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

func toInt(value any) (int, bool) {
	switch n := value.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		f := float64(n)
		if f != float64(int(f)) {
			return 0, false
		}
		return int(f), true
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

func isInt(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isNumber(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func toFloat64(value any) (float64, bool) {
	switch n := value.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

var _ rule.Configurable = (*Rule)(nil)
