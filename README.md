# tidymark

A Markdown linter written in Go.

## Installation

```bash
go install github.com/jeduden/tidymark@latest
```

## Usage

```bash
tidymark <file.md>
```

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/)

### Lint

```bash
golangci-lint run
```

### Test

```bash
go test ./...
```

## License

[MIT](LICENSE)
