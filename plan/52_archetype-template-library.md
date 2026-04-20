---
id: 52
title: Archetype / Template Library for Agentic Patterns
status: "✅"
---
# Archetype / Template Library for Agentic Patterns

## Goal

Provide a richer template system with pre-built archetypes
for common agentic Markdown artifacts.

## Tasks

1. [x] Define supported archetype metadata and
   required-structure enforcement rules.
2. [x] Add archetype templates for common agent patterns
   (story-file, prd, agent-definition, claude-md).
3. [x] Update required-structure rule
   to validate archetype selection and required sections.
4. [x] Document archetype usage and configuration.

## Acceptance Criteria

- [x] Archetype templates are available for common agentic document types.
- [x] Required-structure rule validates selected archetype sections.
- [x] Documentation explains how to add and configure archetypes.
- [x] All tests pass: `go test ./...`
- [x] `golangci-lint run` reports no issues
