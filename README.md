# PanCtx

Markdown-Pandoc-ConTEXt-PDF converter

This utility converts markdown files to pdf with Pandoc and Context. 

## Installation

To install a binary release:

- download the file matching your platform here: [Latest release
  binaries](https://github.com/adnsv/panctx/releases/latest)
- unzip it into the directory of your choice
- make sure your system path resolves to that directory

To build and install PanCtx from sources:

- make sure you have a recent GO compiler installed
- execute `go install github.com/adnsv/panctx@latest`

## Running PanCtx

PanCtx is a command line utility that can be executed from a terminal or from a script.

A temporary scratch dir on your system is required to store some intermediate files. The location of this directory must be specified with `-w=<WORKDIR>` or `--workdir=<WORKDIR>` command line option.

A location of a template file (see below) for document generation can be specified with `-t=<TEMPLATE-FILE>` or `--template=<TEMPLATE-FILE>` option. This flag is required when generating PDF files, without it PanCtx will only generate `.tex` files in `ConTEXt` format without producing any PDFs.

By default, generated PDF file is saved inside the working directory. Using `-o=<OUTPUT-FILE>` or `--output=<OUTPUT-FILE>` you can rename and move it to the location of your choice.

Defitions and overrides to template definitions can be specified with `-d` or `--def` parameter followed by a `name=value` pair.

The command line arguments must specify a main input file. The input is a ConTEXt flavored `.tex` file that contains lins and references to template assets.

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

- `top-heading`: controls mapping of level one markdown headings to the generated ConTEXt headings. Supported values are `part`, `chapter`, `section`. Default is `section`.

- `default-externalfigure-size`: can be used when importing external images (except for .svg). External figures are mapped to `\externalfigure[...]` statements in context. If the corresponding markdown does has no size constraints (e.g. no `width` and no `height` attributes), then the statement from `default-externalfigure-size` will be injected.

Notice also, that mapping of markdown descriptions, requires a custom ConText definition:

```tex
\definedescription[description][
  alternative=top, 
  headstyle=\ss\bf, 
  margin=2em,
  inbetween=,
  headcommand=\descriptionHeadCommand
]
```

TODO: more detailed description