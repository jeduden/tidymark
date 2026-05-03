---
summary: >-
  How to use the build directive to declare artifact outputs, keep
  generated bodies in sync, and configure built-in and user recipes.
---
# Build directive

The `<?build?>` directive declares a build artifact — a file
produced from a recipe (e.g. a screenshot or a VHS terminal
recording). `mdsmith fix` renders the section body from the
recipe's `body_template` and keeps it up to date. No external
tool runs at lint time.

## Syntax

```text
<?build
recipe: RECIPE-NAME
output: path/to/artifact.ext
[recipe-specific params]
?>
RENDERED BODY
<?/build?>
```

The directive uses the same block form as `<?catalog?>` and
`<?include?>`. Inline form is not supported.

### Common parameters

| Name     | Required | Description                                                     |
|----------|----------|-----------------------------------------------------------------|
| `recipe` | yes      | Built-in or user-declared recipe name                           |
| `output` | yes      | Artifact path relative to the Markdown file; no `..` components |

`output` accepts any file extension; the rule applies no
extension filter.

## Built-in recipes

### `screenshot`

Captures a browser screenshot. Parameters:

| Param      | Required | Default    | Description              |
|------------|----------|------------|--------------------------|
| `url`      | yes      | —          | Page URL to capture      |
| `selector` | no       | full page  | CSS selector to capture  |
| `viewport` | no       | `1280x800` | Viewport size            |
| `wait`     | no       | `0` ms     | Wait time before capture |
| `click`    | no       | —          | CSS selector to click    |
| `hide`     | no       | `[]`       | CSS selectors to hide    |

#### Screenshot example

```text
<?build
recipe: screenshot
url: /dashboard
output: docs/dashboard.png
?>
![screenshot output: docs/dashboard.png](docs/dashboard.png)
<?/build?>
```

### `vhs`

Records a terminal session using [VHS](https://github.com/charmbracelet/vhs).
Parameters:

| Param   | Required | Description              |
|---------|----------|--------------------------|
| `input` | yes      | Path to the `.tape` file |

#### VHS example

```text
<?build
recipe: vhs
input: demo.tape
output: demo.gif
?>
![vhs output: demo.gif](demo.gif)
<?/build?>
```

## Generated body

`mdsmith fix` renders the section body from the recipe's
`body_template`. Two placeholders are available:

| Placeholder | Value                                   |
|-------------|-----------------------------------------|
| `{output}`  | The `output` param value                |
| `{alt}`     | `"{recipe} output: {output}"` (default) |

Default `body_template` values:

| Recipe       | Default `body_template` |
|--------------|-------------------------|
| `screenshot` | `![{alt}]({output})`    |
| `vhs`        | `![{alt}]({output})`    |
| custom       | `[{output}]({output})`  |

The rendered alt text from built-in recipes satisfies
[MDS032](../../reference/cli/check.md) (non-empty alt text).

## User-declared recipes

Declare custom recipes in `build.recipes` in `.mdsmith.yml`:

```yaml
build:
  recipes:
    chart:
      body-template: "![{alt}]({output})"
      params:
        required: [data]
        optional: [title]
```

Then reference the recipe in a `<?build?>` directive:

```text
<?build
recipe: chart
data: revenue.csv
output: docs/revenue-chart.png
?>
![chart output: docs/revenue-chart.png](docs/revenue-chart.png)
<?/build?>
```

If `body-template` is omitted, the default `[{output}]({output})`
is used.

## Rule MDS039

MDS039 validates `<?build?>` directives and reports:

- **Error** when `recipe` is missing or unknown
- **Error** when `output` is missing or contains `..` components
- **Error** when a required param for the recipe is absent
- **Warning** when a param is not in the recipe's `required` or
  `optional` lists
- **Error** (`generated section is out of date`) when the body
  diverges from the rendered `body_template`

Run `mdsmith fix <file>` to regenerate stale bodies.

## Interaction with other rules

- **MDS027**: a missing artifact file fires MDS027 independently;
  MDS039 does not duplicate it.
- **MDS032**: the default `body_template` for built-in recipes
  always produces non-empty alt text.
- **merge-driver**: regenerates `<?build?>` bodies on conflict
  via `gensection.Engine`; artifact bytes are not regenerated.
