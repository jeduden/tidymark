---
id: MDS005
name: no-duplicate-headings
status: ready
description: No two headings should have the same text.
---
# MDS005: no-duplicate-headings

No two headings should have the same text.

- **ID**: MDS005
- **Name**: `no-duplicate-headings`
- **Status**: ready
- **Default**: enabled
- **Fixable**: no
- **Implementation**:
  [source](./)
- **Category**: heading

## Config

Enable:

```yaml
rules:
  no-duplicate-headings: true
```

Disable:

```yaml
rules:
  no-duplicate-headings: false
```

## Examples

### Bad

```markdown
# Introduction

## Overview

## Overview
```

### Good

```markdown
# Introduction

## Project Overview

## API Overview
```
