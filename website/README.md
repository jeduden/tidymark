---
summary: mdsmith.dev — the docs-and-marketing site, built with Hugo from the repository's existing Markdown.
---
# mdsmith.dev

The single product surface that complements the `mdsmith` CLI:
homepage, docs, rules catalog, comparison page. Source content
is the repository itself — `README.md` and `docs/**/*.md` — so
the website cannot drift out of sync with the binary.

## Layout

| Path                           | Purpose                                                              |
|--------------------------------|----------------------------------------------------------------------|
| `hugo.toml`                    | Site config + module mounts; pulls `../docs` into `content/docs/`.   |
| `content/_index.md`            | Homepage front matter and copy.                                      |
| `content/compare.md`           | "vs other linters" page.                                             |
| `content/docs/`                | **Mounted** from the repository's `docs/` tree — do not author here. |
| `layouts/_default/baseof.html` | Page shell — `<head>`, top nav, footer.                              |
| `layouts/index.html`           | Homepage template (hero · feature grid · install tabs · terminal).   |
| `layouts/_default/single.html` | Docs page template (sidebar + prose).                                |
| `layouts/_default/list.html`   | Section index template.                                              |
| `layouts/partials/`            | `topnav`, `footer`, `hero`, `feature-grid`, etc.                     |
| `layouts/shortcodes/`          | `callout`, `diag`, `pill`, `chip`, `install-cmd`.                    |
| `layouts/_default/_markup/`    | Goldmark render hooks (headings, code blocks).                       |
| `static/css/`                  | `colors_and_type.css` (tokens) + `app.css` (component styles).       |
| `static/fonts/`                | 0xProto Nerd Font TTF.                                               |
| `static/img/`                  | Logo SVGs.                                                           |
| `data/`                        | Reserved for generated data (rules catalog, etc.).                   |

## Develop

Hugo (non-extended is fine — no SCSS or image processing):

```bash
# install hugo
go install github.com/gohugoio/hugo@latest

# sync ../docs/ -> content/docs/ (mdsmith fix + escape Hugo
# shortcodes + drop schema templates + promote index.md to
# _index.md). content/docs/ is .gitignore'd.
cd website && ./scripts/sync-docs.sh

# build into public/
hugo --minify

# or serve with live reload (re-run sync-docs after editing docs/)
hugo server -D
```

## Source layout

`docs/**/*.md` is the single source of truth. The website
never owns docs content. `scripts/sync-docs.sh` snapshots
it into `content/docs/`, applies the transforms Hugo needs
(escape `{{< … >}}`, drop `proto.md` schema templates,
rename `index.md` → `_index.md`, prune non-content files),
and `mdsmith fix` materializes every `<?catalog?>` and
`<?include?>` body before the copy happens. Pass
`--no-fix` to skip the `mdsmith fix` step.

## Deployment

Tag-driven GitHub Pages publish lives in the release workflow
(see `docs/development/release.md`). The site version always
matches the binary release on the tag.

## Design system origin

The CSS in `static/css/` is the Claude Design export for the
project. See:

- `static/css/colors_and_type.css` — tokens, base typography
- `static/css/app.css` — component styles (class-based, no
  React or Tailwind required)
- `static/img/logo-*.svg` — substitution logos; replace if
  mdsmith adopts an official mark

The CSS classes (`.topnav`, `.hero`, `.feature-grid`,
`.install`, `.term`, `.diag`, `.docs-grid`, `.chip`, `.pill`,
`.pg`, `.tbl`, `.footer`, …) drive the partials and shortcodes
in `layouts/`. To restyle, edit the CSS; markup stays put.
