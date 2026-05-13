---
title: Go SOLID and clean architecture patterns
summary: >-
  Language-level SOLID and clean architecture
  rules for Go codebases. Project-agnostic.
---
# Go SOLID and clean architecture patterns

Language-level rules. The project's own
docs (commonly under
`docs/development/architecture/`) describe
how the project instantiates these patterns
with named packages.

## Single responsibility per package

A Go package is the unit of responsibility.
Name a package after the one question it
answers; keep it small enough that the
answer fits in one sentence.

Examples (generic):

- `config` answers "what does the user's
  configuration say?"
- `lint` answers "does this file pass
  every rule?"
- `fix` answers "what edits make it pass?"

A package named `util`, `common`,
`helpers`, `lib`, or `misc` is almost
always an SRP violation: it answers "a
grab bag of things". Split it by question.

When a function is needed by two callers,
ask: does it belong in the lower of the two
packages, or in its own package named for
the question it answers? "Lift it into a
new helper" is rarely the right answer; in
Go the gravity of a single-question
package is what keeps the graph clean.

## Open/closed via plugin packages

The core engine of a system should not
change when a new variant (rule, encoder,
strategy) is added. The contract:

- A small "ports" package defines the
  interfaces — `Rule`, `Encoder`, etc.
- Each variant lives in its own package
  and implements the interface.
- A central blank-import barrel registers
  every production variant in `init()` so
  consumers reach a populated registry
  without listing the variants
  themselves.

When you add a new variant, create the
package, implement the interface, and add a
blank-import line in the barrel. The engine
remains untouched.

If the new variant needs the engine to
expose new data, extend the interface in
the ports package — not the engine, and
not from the variant by reaching upward.

## Liskov: every implementation is interchangeable

Recurring pitfalls:

1. An implementation that only works for
   certain inputs (e.g. only files of a
   certain kind). The selection belongs in
   the call site or in config; the
   implementation sees what it is fed and
   honors the contract on every input.
2. An implementation that panics on edge
   cases the caller considers valid (empty
   input, deep nesting, generated regions).
   Return an error or a no-op instead.

If an implementation cannot honor the
contract under some condition, that
condition belongs in filtering at the
caller. Do not hide a bail-out inside the
implementation.

## Interface segregation

Define small interfaces with one question
each. A `Rule` interface answers "did this
file pass?"; a separate `Fixer` interface
answers "what edits would fix it?". A rule
implements only the interfaces it satisfies;
callers type-assert when they need a
capability.

```go
if f, ok := r.(Fixer); ok {
    edits = f.Fix(file)
}
```

Do not add a method to a base interface to
serve one implementation. Define a new
interface, let the implementation satisfy
both, and have the caller type-assert.

## Dependency inversion across layers

Go enforces dependency direction at
compile time via imports. Use that as the
boundary:

- Entry points (`cmd/<binary>/`) may
  import everything below.
- Mid-layer orchestrators may import their
  ports package, never the leaves that
  implement those ports.
- Variant packages may import the ports
  package and shared low-level helpers
  (text parsing, file walking) but never
  the orchestrator.
- Pure helpers at the bottom never import
  anything from above.

A circular-import error from `go build` is
often the first sign of an inversion
violation. Do not break the cycle by moving
a type; break it by inverting the
dependency through an interface.

## Clean architecture in `cmd/`

`cmd/<binary>/` is wiring only:

- Parse flags.
- Construct the orchestrator with its
  dependencies.
- Invoke a subcommand handler.
- Translate the result into an exit code
  and output stream.

Domain logic that belongs in a lower
package should not leak into `cmd/`. A
handler in `cmd/` longer than ~50 lines is
a smell.

## Errors and panics

- Error messages: lowercase, no trailing
  punctuation (standard Go style).
- Wrap with `fmt.Errorf("...: %w", err)`;
  the caller decides how to surface.
- Reserve `panic` for invariants that, if
  violated, indicate a programming bug —
  impossible enum case, internal cache
  invariant. Never panic on user input.
- Don't define a typed error whose only
  field is a string; reuse `errors.New` or
  `fmt.Errorf`. Define typed errors only
  when callers will inspect them.

## Testing patterns

- Unit tests next to the package
  (`xxx_test.go`).
- Cross-package tests in a sibling
  `integration` package, kept small.
- Variant fixtures live alongside the
  variant (e.g. `<variant>/good/`,
  `<variant>/bad/`). Good fixtures pass
  every default-enabled rule; bad ones are
  excluded from the production lint set.
- Use `testify/require` for preconditions
  that should abort the test;
  `testify/assert` for soft checks.
- Use `Same`/`NotSame` for pointer
  identity.
- Do not mock at the ports-package
  boundary. Mocks at a boundary suggest the
  boundary is wrong; feed real inputs via
  fixtures.

## Common violations to flag

- A package outside the variant tree
  importing a specific variant package.
- A type defined in the orchestrator that
  is the public API for variants — should
  live in the ports package.
- A function in the orchestrator called
  only by one variant — move it down.
- A config field consumed only by one
  variant — move it to that variant's
  settings struct.
- A test that imports the orchestrator to
  test a variant — push it down to a
  fixture instead.
- A `Helper`, `Util`, or `Misc` symbol
  anywhere. The name is the problem.

## Refactor moves that usually work

- Push a leaky abstraction down. A type
  defined in a high layer but consumed
  only in low layers belongs in the low
  layer.
- Lift a shared dependency up to an
  interface. Two implementations needed in
  the same place → define an interface and
  inject it.
- Split a package by question. A package
  whose top-level doc comment needs "and"
  to describe should become two.
- Replace a `switch` on type with method
  dispatch. A function in the orchestrator
  that switches on variant ID belongs as a
  method on the variant interface.
