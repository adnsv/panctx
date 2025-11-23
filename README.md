# PanCtx

Markdown-Pandoc-ConTEXt-PDF converter

PanCtx is a command-line tool that converts Markdown documents to professionally typeset PDFs using a template-based approach. It leverages Pandoc for Markdown parsing and ConTeXt for PDF generation, supporting GitHub-style alerts, custom layouts, and extensive formatting options.

## Prerequisites

Before installing PanCtx, ensure you have the following tools installed:

- **Pandoc** (required): For converting Markdown to JSON AST
  - Installation: https://pandoc.org/installing.html
  - Verify: `pandoc --version`

- **ConTeXt** (required for PDF generation): For typesetting and PDF creation
  - Installation: https://wiki.contextgarden.net/Installation
  - Verify: `context --version`

- **Go compiler** (optional, for building from source): Version 1.16 or later
  - Installation: https://golang.org/doc/install
  - Verify: `go version`

## Installation

To install a binary release:

- download the file matching your platform here: [Latest release
  binaries](https://github.com/adnsv/panctx/releases/latest)
- unzip it into the directory of your choice
- make sure your system path resolves to that directory

To build and install PanCtx from sources:

- make sure you have a recent GO compiler installed
- execute `go install github.com/adnsv/panctx@latest`

## Quick Start

Basic usage to convert a Markdown document to PDF:

```bash
panctx -w=./workdir -t=template.yaml main.tex
```

Where:
- `main.tex`: ConTeXt file that references your Markdown content
- `template.yaml`: Template configuration with variables and assets
- `./workdir`: Temporary directory for intermediate files

Example `main.tex`:

```tex
\input $<template:preamble.tex>$

\startdocument[$<var:title>$]
\startbodymatter
\input $<markdown:content.md>$
\stopbodymatter
\stopdocument
```

See detailed examples in the sections below.

## Running PanCtx

PanCtx is a command line utility that can be executed from a terminal or from a script.

A temporary scratch dir on your system is required to store some intermediate files. The location of this directory must be specified with `-w=<WORKDIR>` or `--workdir=<WORKDIR>` command line option.

A location of a template file (see below) for document generation can be specified with `-t=<TEMPLATE-FILE>` or `--template=<TEMPLATE-FILE>` option. This flag is required when generating PDF files, without it PanCtx will only generate `.tex` files in `ConTEXt` format without producing any PDFs.

By default, generated PDF file is saved inside the working directory. Using `-o=<OUTPUT-FILE>` or `--output=<OUTPUT-FILE>` you can rename and move it to the location of your choice.

Definitions and overrides to template definitions can be specified with `-d` or `--def` parameter followed by a `name=value` pair.

The command line arguments must specify a main input file. The input is a ConTEXt flavored `.tex` file that contains links and references to template assets.

## Sample Main File

Main file defines the overall structure of the document. Here is an example:

```tex
\input $<template:preamble.tex>$

\startdocument[$<var:title>$]

\startfrontmatter
\input $<template:front-page.tex>$
\input $<template:legal-page.tex>$
\input $<template:contents.tex>$
\stopfrontmatter

\startbodymatter
\input $<markdown:fulltext.md>$
\stopbodymatter

\stopdocument
```

This is a ConTEXt flavored tex format with placeholders referring to template variables and files. References to markdown sources are processed by PanCtx using markdown->json->context filters.

## Template File

Template file provides definitions for variables and declares asset files. Variables can be referenced in other places as `$<var:name>$`. Asset files can be referenced as `$<template:name>$`.

Here is an example:

```yml
def:
  fontsize: 12pt
  top-heading: chapter
  default-externalfigure-size: width=0.9\textwidth

layouts:
  A4: backspace=63pt,width=468pt,topspace=49pt,height=744pt
  letter: backspace=72pt,width=468pt,topspace=24pt,height=744pt

assets:
  - logo.svg
  - front-page.tex
  - legal-page.tex
  - contents.tex
  - preamble.tex
```

Note that variable definitions declared in the template can be overriden in the command line with `-d` or `--def` flags.

Page size and layout variables have special handling:

- `pagesize` specifies the size of the page for layout purposes, typical values are `A4`, `letter` (default), etc.
- `papersize` specifies the size of paper for printing, which can be used to generate a PDF file with multiple pages per sheet. If not explicitly defined, `papersize` defaults to be the same as `pagesize`.
- `layout` specifies margins and other page layout settings.
- `layouts` allows to specify a layout for each pagesize. This layout gets automatically loaded into the `layout` variable.

To support this behavior, include the following fragment in your `preamble.tex` or into your main `.tex` document:

```tex
\setupbodyfont[mainface, $<var:fontsize>$]
\setuppapersize[$<var:pagesize>$][$<var:papersize>$]
\setuplayout[$<var:layout>$]
```

There is also a couple of special definitions that control generated content:

- `top-heading`: controls mapping of level one markdown headings to the generated ConTeXt headings. Supported values are `part`, `chapter`, `section`. Default is `chapter`.

- `default-externalfigure-size`: can be used when importing external images (except for .svg). External figures are mapped to `\externalfigure[...]` statements in ConTeXt. If the corresponding markdown has no size constraints (e.g. no `width` and no `height` attributes), then the statement from `default-externalfigure-size` will be injected.

Notice also, that mapping of markdown descriptions requires a custom ConTeXt definition:

```tex
\definedescription[description][
  alternative=top, 
  headstyle=\ss\bf, 
  margin=2em,
  inbetween=,
  headcommand=\descriptionHeadCommand
]
```

## Markdown Features

PanCtx uses Pandoc to convert Markdown to JSON AST, then converts the AST to ConTeXt format. The following features are supported:

### Standard Markdown

- **Headings**: Markdown headings (levels 1-6) are mapped to ConTeXt sectioning commands
- **Paragraphs**: Standard paragraphs and line blocks
- **Emphasis**: *italic*, **bold**, ~~strikethrough~~, superscript, subscript, small caps
- **Lists**: Ordered lists, bullet lists, and definition lists
- **Code**: Inline code and fenced code blocks with syntax highlighting
- **Links**: Hyperlinks and cross-references (links ending with `#` become `\in` references)
- **Images**: Inline images and floating figures with captions
- **Tables**: Full table support using ConTeXt's xtable system
- **Blockquotes**: Standard blockquotes and GitHub alerts (see below)
- **Math**: Inline and display math using LaTeX syntax
- **Raw ConTeXt**: Raw ConTeXt code can be embedded with ` ```{=tex} ` blocks

### GitHub Alerts

PanCtx supports GitHub-style alerts using blockquote syntax:

```markdown
> [!NOTE]
> Helpful information for users

> [!TIP]
> Helpful advice or suggestions

> [!IMPORTANT]
> Critical information users need to know

> [!WARNING]
> Urgent information requiring attention

> [!CAUTION]
> Potential negative consequences of an action
```

Each alert type is converted to a corresponding ConTeXt environment (`\startNOTE`, `\startTIP`, etc.). To style these alerts, define the corresponding framed text environments in your preamble:

```tex
\defineframedtext[NOTE][
  width=broad,
  background=color,
  backgroundcolor=...,
  leftframe=on,
  framecolor=...,
  leftframethickness=4pt,
  ...
]
```

### Special Div Classes

PanCtx supports special div classes for layout control:

- `HSTACK`: Horizontal layout using table cells (separated by horizontal rules)
- `narrower=<amount>`: Narrower text block
- `combination=<spec>`: ConTeXt combination environment
- `columns=<spec>`: Multi-column layout

### Image Attributes

Images support special attributes:

- `width`, `height`: Size constraints (supports %, px, cm, mm, in, pt, em)
- `placement=inline`: Forces inline placement (prevents floating figure)
- `dx`, `dy`: Offset positioning
- `options`: Additional ConTeXt figure options