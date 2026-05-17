---
title: "Rename without breaking links"
summary: >-
  Rename a heading and every workspace anchor link that points at it
  is rewritten in one atomic edit. Link-reference labels rename with
  their uses. A colliding slug fails loudly instead of silently
  breaking cross-file links.
icon: replace
weight: 17
link: "/docs/reference/cli/lsp/"
---
# Rename without breaking links

Renaming a heading normally breaks every `](file.md#old-slug)`
that pointed at it. The links still parse, so nothing
complains until a reader hits a dead anchor — or until
MDS027 flags it on the next lint pass, after the damage is
committed.

mdsmith renames the whole graph at once. Rename a heading
and the editor rewrites the heading line plus every
workspace anchor link that resolved to its slug, in a
single atomic edit. Same-file `(#slug)` references are
included. When a duplicate-name disambiguator shifts —
renaming the first "Setup" changes the second's slug from
`setup-1` to `setup` — the affected links update too.

Link-reference labels rename the same way. The `[label]:
url` definition and every `[text][label]` and shortcut
`[label]` use in the file move together.

The rename refuses to corrupt the workspace. If the new
heading text slugifies to a slug another heading already
owns, the rename fails and names the colliding heading
rather than silently shifting numbered suffixes. A label
that collides with another definition fails the same way.
Text that slugifies to nothing, or that contains a newline
or a stray bracket, is rejected before any edit applies.

Today this is the LSP `rename` capability: any LSP-aware
editor, and the Claude Code agent, can drive it. See the
[LSP reference](../reference/cli/lsp.md) for the
prepare-range table and the collision-error contract.
