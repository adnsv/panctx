// Package main provides the panctx command-line tool for converting Markdown documents
// to PDF via Pandoc and ConTeXt. It supports template-based document generation with
// variable substitution and GitHub-style alerts.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/adnsv/panctx/context"
	cli "github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("panctx", "Pandoc->ConTeXt converter")

	app.Version("version", app_version())

	app.Spec = "-w=<WORKDIR> [-t=<TEMPLATE-FILE>] [-d=<var=value>] [-o=<OUTPUT-FILE>] INPUT"

	mainInputFN := ""
	workdir := ""
	templateFN := ""
	outFN := ""
	definitions := []string{}

	app.StringOptPtr(&workdir, "w workdir", "", "a directory for temporary files")
	app.StringOptPtr(&templateFN, "t template", "", "specify a yaml template file (required for PDF generation)")
	app.StringsOptPtr(&definitions, "d def", nil, "add definition")
	app.StringOptPtr(&outFN, "o output", "", "output filename for the generated PDF file (also requires -t flag)")
	app.StringArgPtr(&mainInputFN, "INPUT", "", "input file")

	app.Action = func() {
		log.Printf("using workdir %s\n", workdir)
		workdir, err := filepath.Abs(workdir)
		if err != nil {
			log.Fatal(err)
		}
		err = os.MkdirAll(workdir, 0755)
		if err != nil {
			log.Fatal(err)
		}

		prj := context.NewProject(workdir)

		prj.Definitions["fontsize"] = "12pt"
		prj.Definitions["pagesize"] = "letter"
		prj.Definitions["title"] = ""
		prj.Definitions["subtitle"] = ""
		prj.Definitions["date"] = ""

		err = prj.LoadConfig(templateFN)
		if err != nil {
			log.Fatal(err)
		}
		err = prj.LoadMain(mainInputFN)
		if err != nil {
			log.Fatal(err)
		}

		for _, def := range definitions {
			kv := strings.SplitN(def, "=", 2)
			if len(kv) != 2 {
				log.Fatalf("invalid definition %s", def)
			}
			prj.Definitions[strings.TrimSpace(kv[0])] = kv[1]
		}

		if v, ok := prj.Layouts[prj.Definitions["pagesize"]]; ok {
			prj.Definitions["layout"] = v
		}
		if _, ok := prj.Definitions["papersize"]; !ok {
			prj.Definitions["papersize"] = prj.Definitions["pagesize"]
		}

		err = prj.Process()
		if err != nil {
			log.Fatal(err)
		}

		pdfFN, err := prj.BuildPDF()
		if err != nil {
			log.Fatal(err)
		}

		if outFN != "" {
			log.Printf("moving generated PDF to: %s\n", outFN)
			err = os.Rename(pdfFN, outFN)
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Printf("mission accomplished\n")

	}

	app.Run(os.Args)
}
