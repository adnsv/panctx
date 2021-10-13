# PanCtx
Markdown-Pandoc-ConTEXt-PDF filter

This little utility effectively converts markdown files to pdf. 

## Obtaining PanCtx

Prerequisite: make sure GO language compiler is installed on your system.

Downloading source code:

```sh
go get github.com/adnsv/panctx
```

Updating source code:

```sh
go get -u github.com/adnsv/panctx
```

After the source code for PanCtx is obtained, you can execute it directly:

```sh
go run github.com/adnsv/panctx <args...>
```

Alternatively, build and install it into GO user path with the following command:

```sh
go install github.com/adnsv/panctx
```

If your user's GO path is configured correctly, you should be able to run this utility as a regular executable:

```sh
panctx <args...>
```

## Running PanCtx

PanCtx is a command line utility that can be executed from a terminal or from a script.

A temporary scratch dir on your system is required to store some intermediate files. The location of this directory must be specified with `-w=<WORKDIR>` or `--workdir=<WORKDIR>` command line option.

A location of a template file (see below) for document generation can be specified with `-t=<TEMPLATE-FILE>` or `--template=<TEMPLATE-FILE>` option. This flag is required when generating PDF files, without it PanCtx will only generate `.tex` files in `ConTEXt` format without producing any PDFs.

By default, generated PDF file is saved inside the working directory. Using `-o=<OUTPUT-FILE>` or `--output=<OUTPUT-FILE>` you can rename and move it to the location of your choice.

Defitions and overrides to template definitions can be specified with `-d` or `--def` parameter followed by a `name=value` pair.

The command line arguments must specify a main input file. The input is a ConTEXt flavored `.tex` file that contains lins and references to template assets.

## Sample Main File

A typical main file defines the overall structure of the document. Here is an example:

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

Note, that variable definitions declared in the template can be overriden in the command line with `-d` or `--def` flags.

Page size and layout variables have special handling:

- `pagesize` specifies the size of the page for layout purposes, typical values are `A4`, `letter` (default), etc.
- `papersize` specifies the size of paper for printing, which can be used to generate a PDF file with multiple pages per sheet. If not explicitly defined, `papersize` defaults to be the same as `pagesize`.
- `layout` specifies margins and other page layout settings.
- `layouts` allows to specify a layout for each pagesize. This layout gets automatically loaded into the `layout` variable.



TODO: more detailed description