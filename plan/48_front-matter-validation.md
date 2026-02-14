---
id: 48
title: Front Matter Validation
status: ✅
template:
  allow-extra-sections: true
---
# Front Matter Validation

## Goal

Validate YAML front matter in Markdown files against a schema
to prevent silent metadata breakage in agent workflows.

## Tasks

1. Define schema format and configuration for required fields,
   types, and allowed values.
2. Parse front matter safely and report missing or invalid fields
   with actionable messages.
3. Integrate rule with existing file discovery and rule registry.
4. Document schema options and examples in rule docs.

## Acceptance Criteria

- [x] Rule fails when required front matter fields are missing.
- [x] Rule fails when field types or allowed values are invalid.
- [x] Errors include file path and field name.
- [x] All tests pass: `go test ./...`
- [x] `golangci-lint run` reports no issues
