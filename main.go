package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/adnsv/panctx/context"
	modelv2 "github.com/adnsv/panctx/model-v2"
	cli "github.com/jawher/mow.cli"
)

func main() {
	mainInputFN := ""
	workdir := ""
	templateFN := ""
	outFN := ""
	definitions := []string{}

	app := cli.App("panctx", "Pandoc->ConTeXt converter")
	app.Spec = "-w=<WORKDIR> [-t=<TEMPLATE-FILE>] [-d=<var=value>] [-o=<OUTPUT-FILE>] INPUT"
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

		mainExt := strings.ToLower(filepath.Ext(mainInputFN))
		if mainExt == ".yml" || mainExt == ".yaml" {
			tt, err := modelv2.LoadTargets(mainInputFN)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("loaded %d targets\n", len(tt))
			fmt.Printf("done\n")
			return
			//log.Fatal("yaml files are not supported yet as main files")
			//fmt.Printf("loaded: %s", proj.Title)
		}

		prj := context.NewProject(workdir)

		prj.Definitions["fontsize"] = "12pt"
		prj.Definitions["pagesize"] = "letter"
		prj.Definitions["title"] = ""
		prj.Definitions["subtitle"] = ""
		prj.Definitions["date"] = ""
		err = prj.LoadTemplate(templateFN)
		if err != nil {
			log.Fatal(err)
		}

		err = prj.LoadMainTex(mainInputFN)
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
