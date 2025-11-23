# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PanCtx is a command-line utility that converts Markdown files to PDF through a Pandoc → ConTEXt → PDF pipeline. It uses a template-based approach with variable substitution to produce professionally typeset documents.

## Building and Running

### Build from source
```bash
go build
```

### Install from source
```bash
go install github.com/adnsv/panctx@latest
```

### Run the application
```bash
./panctx -w=<WORKDIR> -t=<TEMPLATE-FILE> [-d=<var=value>] [-o=<OUTPUT-FILE>] <INPUT>
```

Required flags:
- `-w` or `--workdir`: Temporary scratch directory for intermediate files
- `-t` or `--template`: YAML template file (required for PDF generation; without it only .tex files are generated)

Optional flags:
- `-d` or `--def`: Define/override template variables (format: `name=value`)
- `-o` or `--output`: Output filename for generated PDF (default: saves in workdir)

### Dependencies

Runtime dependencies (external):
- `pandoc`: For converting markdown to JSON AST
- `context`: For generating PDF from ConTEXt .tex files

Go module dependencies:
- `github.com/adnsv/go-pandoc`: Go bindings for Pandoc JSON AST
- `github.com/adnsv/go-utils`: Utilities including filesystem helpers
- `github.com/jawher/mow.cli`: CLI argument parsing
- `gopkg.in/yaml.v3`: YAML parsing for template files

## Architecture

### Core Processing Pipeline

1. **Configuration Loading** (`context/proj.go:LoadConfig`):
   - Parses YAML template file containing variable definitions, layouts, and asset paths
   - Template assets (`.tex`, `.svg`, etc.) are registered for copying to workdir

2. **Main File Loading** (`context/proj.go:LoadMain`):
   - Reads main ConTEXt `.tex` file
   - Scans for `$<type:reference>$` placeholders
   - For markdown references (`$<markdown:file.md>$`):
     - Converts markdown to JSON using `pandoc -t json`
     - Parses JSON into Pandoc AST
     - Extracts metadata from markdown frontmatter into project definitions

3. **Processing Phase** (`context/proj.go:Process`):
   - Replaces all `$<...>$` placeholders in main file and template assets:
     - `$<var:name>$`: Variable from definitions
     - `$<template:file.tex>$`: Template asset path
     - `$<markdown:file.md>$`: Markdown asset path
   - For each markdown asset, converts Pandoc AST to ConTEXt using custom writer
   - Writes all processed files to workdir

4. **PDF Generation** (`context/proj.go:BuildPDF`):
   - Executes `context` command on processed main .tex file
   - Returns path to generated PDF

### Placeholder System

Three types of placeholders in input files (format: `$<type:reference>$`):

- **`var`**: Substitutes variable from `Definitions` map (escaped for ConTEXt)
- **`template`**: Resolves to workdir path of template asset
- **`markdown`**: Resolves to workdir path of converted markdown asset

### Markdown → ConTEXt Conversion

The `context/writer.go` module implements a custom Pandoc AST → ConTEXt writer:

- **Block elements**: Maps Pandoc blocks (paragraphs, lists, tables, code blocks, etc.) to ConTEXt equivalents
  - Tables use ConTEXt's xtable system
  - Divs with special classes enable layout features (HSTACK, narrower, combination, columns)
  - Admonitions detected via `**!Heading**` or `**!!Heading**` syntax → NOTE/WARNING environments

- **Inline elements**: Handles formatting (bold, italic, code, math), images, links
  - Images can be inline (`placement=inline`) or floating figures
  - SVG images handled specially (no default size constraint)
  - Links ending with `#` are converted to internal cross-references (`\in`)

- **Heading levels**: Configurable top-level division via `top-heading` variable:
  - `part`: Level 1 headings → `\part`
  - `chapter`: Level 1 headings → `\chapter` (default)
  - `section`: Level 1 headings → `\section`

### Special Template Variables

Default definitions (main.go:45-49):
- `fontsize`: Default `12pt`
- `pagesize`: Default `letter`
- `title`, `subtitle`, `date`: Empty by default, populated from markdown metadata

Auto-populated variables:
- `layout`: Auto-selected from `layouts` map based on `pagesize`
- `papersize`: Defaults to `pagesize` if not explicitly set

Processing-related definitions:
- `top-heading`: Controls markdown heading mapping (see above)
- `default-externalfigure-size`: Applied to external images without explicit dimensions (e.g., `width=0.9\textwidth`)

## Code Structure

### Package Layout

- **`main.go`**: CLI entry point, orchestrates the conversion pipeline
- **`app_version.go`**: Version string handling (supports both `go install` and build-time `-ldflags`)
- **`context/proj.go`**: Project management, configuration loading, file processing orchestration
- **`context/writer.go`**: Pandoc AST to ConTEXt converter

### Key Types

- **`Project`** (`context/proj.go`): Central data structure holding:
  - Directories: `MainDir`, `ConfigDir`, `WorkDir`
  - `Definitions`: Variable map for placeholder substitution
  - `Layouts`: Pagesize-specific layout definitions
  - Asset lists: `MarkdownAssets`, `TemplateAssets`

- **`MarkdownAsset`**: Tracks markdown file through conversion (source path → JSON buffer → Pandoc document → destination .tex path)

- **`TemplateAsset`**: Simple source → destination path mapping for template files

- **`Writer`** (`context/writer.go`): Stateful ConTEXt output generator with context-aware formatting
