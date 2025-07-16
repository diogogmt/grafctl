# grafctl

Manage grafana via the CLI

[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/diogogmt/grafctl)

- [grafctl](#grafctl)
  - [Installation](#installation)
    - [Binary](#binary)
    - [Go](#go)
    - [Homebrew](#homebrew)
  - [Usage](#usage)
    - [Examples](#examples)
  - [Contributing](#contributing)
    - [Makefile](#makefile)

## Installation

#### Binary

For installation instructions from binaries please visit the [Releases Page](https://diogogmt/grafctl/releases).

#### Go

```bash
$ go get diogogmt/grafctl/cmd
```

#### Homebrew

TODO

## Usage

```bash
$ grafctl -h
USAGE
  grafctl [flags] <subcommand>

SUBCOMMANDS
  dash    Manage grafana dashboards
  backup  Backup grafana dashboards and datasources
  import  Import grafana dashboards and datasources

FLAGS
  -key ...        grafana server API key
  -url ...        grafana server API URL
  -verbose false  log verbose output
```

```bash
USAGE
  grafctl dash

SUBCOMMANDS
  ls       List grafana dashboards
  inspect  Inspect grafana dashboard
  sync     sync grafana dashboards
  update-panels-descriptions  update panel descriptions with proper query paths
  export-queries   export panel queries from grafana dashboard to filesystem
```

### Examples

```bash
# backup grafana
$ grafctl -url {{grafana.url}} -key {{api-key}} backup

# restore grafana
$ grafctl -url {{grafana.url}} -key {{api-key}} import ./backup.json.gz

# list dashboards
$ grafctl -url {{grafana.url}} -key {{api-key}} dash ls

# update panel descriptions to include folder, dashboard, row, and panel info
$ grafctl -url {{grafana.url}} -key {{api-key}} dash update-descriptions -uid {{dashboard-uid}}

# preview changes (dry run)
$ grafctl -url {{grafana.url}} -key {{api-key}} dash update-descriptions -uid {{dashboard-uid}} -dry-run

# update all panels, not just those with invalid descriptions
$ grafctl -url {{grafana.url}} -key {{api-key}} dash update-descriptions -uid {{dashboard-uid}} -overwrite
```

#### update-descriptions command

Automatically updates all panel descriptions in a dashboard to a standardized format:

- With row: `query=<folder-title>/<dashboard-title>/<row-title>/<prefix>-<panel-title>`
- Without row: `query=<folder-title>/<dashboard-title>/<prefix>-<panel-title>`

**Options:**
- `-uid` (required): Dashboard UID
- `-overwrite`: Update all panels, not just those with invalid descriptions
- `-dry-run`: Preview changes without updating the dashboard

**Description path segments:**
- Folder title (dashlist/folder)
- Dashboard title
- Row title (if applicable)
- Panel type prefix (e.g., graph, table, stat, etc.)
- Panel title (all segments are kebab-case)

**Example:**
```
query=business-metrics/animation-rate/countries-metrics/graph-viewable-impressions-tiers
# export panel queries from a dashboard
$ grafctl -url {{grafana.url}} -key {{api-key}} dash export-queries -uid {{dashboard-uid}} -out ./queries

# export panel queries and overwrite existing files
$ grafctl -url {{grafana.url}} -key {{api-key}} dash export-queries -uid {{dashboard-uid}} -out ./queries -overwrite
```

## Contributing

#### Makefile

```bash
Usage:

  build         builds grafctl
  install       installs grafctl
  imports       runs goimports
  lint          runs golint
  test          runs go test
  vet           runs go vet
  staticcheck   runs staticcheck
  vendor        updates vendored dependencies
  help          prints this help message
```
