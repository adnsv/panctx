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

The command line arguments must specify one or more input files. Those can be ConTEXt flavored `.tex` files, MarkDown `.md` files or files in other formats supported by Pandoc. All inputs, except for `.tex` will be converted by Pandoc to an intermediate format, then converted to ConTEXt `.tex` files. If the template file is defined, then a ConTEXt pass is executed to convert those into PDFs.

PanCtx supports a few additional flags that allow to override some fields defined within a template:

- `-p=<A4|letter|...>` or `--pagesize=<A4|letter|...>` allows to force a specific page size output (use A4, letter, and other values supported by ConTEXt)

- `--papersize=<A4|letter|...>` similarly to pagesize, this parameter allows to specify custom paper size for PDF generation

- `--top-heading=<section|chapter|part>` allows to customize mapping of markdown headings to ConTEXt document sections.

## Template File

TODO: describe template fields

Here is an example:

```yml
fontsize: 12pt
layouts:
  A4: backspace=63pt,width=468pt,topspace=49pt,height=744pt
  letter: backspace=72pt,width=468pt,topspace=24pt,height=744pt

top-heading: chapter

default-externalfigure-size: width=0.9\textwidth

exec: main # execute this file with context

files:
  - src: main.tex
  - src: logo.svg

  - src: front-page.tex
  - src: legal.tex

front-matter:
  \input front-page
  \input legal

  \setupheadtext[content=Contents]

  \setupcombinedlist[content][list={chapter}, alternative=c,]

  \vfill

  \completecontent

body-matter: 
  \input fulltext
```
