package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adnsv/go-utils/fs"
	"github.com/adnsv/panctx/context"
	"github.com/adnsv/panctx/pandoc"
	cli "github.com/jawher/mow.cli"
)

// var reFileVar = regexp.MustCompile(`(?m)\$<file:(?:.*)>\$`)

func main() {
	inputs := []string{}
	workdir := ""
	templateFN := ""
	outFN := ""
	pageSize := ""
	paperSize := ""
	topHeading := ""

	app := cli.App("panctx", "Pandoc->ConTeXt converter")
	app.Spec = "-w=<WORKDIR> [-t=<TEMPLATE-FILE>] [--pagesize=<A4,letter>] [--top-heading=<section|chapter|part>] [-o=<OUTPUT-FILE>] INPUTS..."
	app.StringOptPtr(&workdir, "w workdir", "", "a directory for temporary files")
	app.StringOptPtr(&templateFN, "t template", "", "specify a yaml template file (required for PDF generation)")
	app.StringOptPtr(&pageSize, "p pagesize", "", "use the specified page size")
	app.StringOptPtr(&paperSize, "papersize", "", "use the specified paper size")
	app.StringOptPtr(&topHeading, "top-heading", "", "treat top-level headings as the given division type")
	app.StringOptPtr(&outFN, "o output", "", "output filename for the generated PDF file (also requires -t flag)")
	app.StringsArgPtr(&inputs, "INPUTS", nil, "input file(s)")

	app.Action = func() {
		files := []string{}
		for _, in := range inputs {
			g, err := filepath.Glob(in)
			if err != nil {
				log.Fatal(err)
			}
			files = append(files, g...)
		}

		err := os.MkdirAll(workdir, 0755)
		if err != nil {
			log.Fatal(err)
		}

		var tmpl *context.Template
		if templateFN != "" {
			if !fs.FileExists(templateFN) {
				log.Fatalf("missing %s", templateFN)
			}
			tmpl, err = context.OpenTemplate(templateFN)
			if err != nil {
				log.Fatal(err)
			}

			// allow overriding some of the template parameters with cli args
			if pageSize != "" {
				tmpl.PageSize = pageSize
			}
			if paperSize != "" {
				tmpl.PaperSize = paperSize
			}
			if topHeading != "" {
				tmpl.TopHeading = topHeading
			}
		}

		type tex struct {
			bn  string
			buf []byte
		}
		type pdoc struct {
			ifn string // input filename
			bn  string // buffer name
			d   *pandoc.Document
		}

		// collect inputs
		texes := []*tex{}
		pdocs := []*pdoc{}
		for _, ifn := range files {
			ext := filepath.Ext(ifn)
			bn := filepath.Base(ifn)
			bn = bn[:len(bn)-len(ext)]
			bn = filepath.Join(workdir, bn)
			log.Printf("loading %s\n", ifn)
			if ext == ".tex" {
				buf, err := os.ReadFile(ifn)
				if err != nil {
					log.Fatal(err)
				}
				texes = append(texes, &tex{bn: bn, buf: buf})
			} else {
				inbuf, err := exec.Command("pandoc", "-t", "json", ifn).Output()
				if err != nil {
					log.Fatal(err)
				}

				d, err := pandoc.NewDocument(inbuf)
				if err != nil {
					log.Fatal(err)
				}

				pdocs = append(pdocs, &pdoc{ifn: ifn, bn: bn, d: d})
			}
		}

		texReplacer := strings.NewReplacer()

		if tmpl != nil {
			for _, t := range pdocs {
				m := t.d.ParseMeta()
				if s, ok := m["title"]; ok {
					if tmpl.Title == "" {
						tmpl.Title = context.FlattenInlines(s)
					} else {
						log.Printf("[warning] found multiple title definitions")
					}
				}
				if s, ok := m["subtitle"]; ok {
					if tmpl.Subtitle == "" {
						tmpl.Subtitle = context.FlattenInlines(s)
					} else {
						log.Printf("[warning] found multiple subtitle definitions")
					}
				}
				if s, ok := m["date"]; ok {
					if tmpl.Date == "" {
						tmpl.Date = context.FlattenInlines(s)
					} else {
						log.Printf("[warning] found multiple date definitions")
					}
				}
			}

			texReplacer = strings.NewReplacer(
				"$<Document.Title>$", tmpl.Title,
				"$<Document.Subtitle>$", tmpl.Subtitle,
				"$<Document.Date>$", tmpl.Date,
			)

			err = tmpl.Execute(workdir, func(b []byte) []byte {
				s := texReplacer.Replace(string(b))
				return []byte(s)
			})
			if err != nil {
				log.Fatal(err)
			}

		}

		for _, t := range texes {
			log.Printf("writing %s\n", t.bn+".tex")

			s := texReplacer.Replace(string(t.buf))

			err = fs.WriteFileIfChanged(t.bn+".tex", []byte(s))
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, t := range pdocs {
			bb, err := t.d.Flow()
			if err != nil {
				log.Fatal(err)
			}

			out := &bytes.Buffer{}
			w := context.NewWriter(out, filepath.Dir(t.ifn))
			if topHeading == "" {
				if tmpl != nil && tmpl.TopHeading != "" {
					topHeading = tmpl.TopHeading
				}
			}
			w.SetTopLevelDivision(topHeading)
			if tmpl != nil {
				w.DefaultExternalFigureSize = tmpl.DefaultExternalFigureSize
			}
			w.WriteBlocks(bb)

			log.Printf("writing %s\n", t.bn+".tex")
			os.WriteFile(t.bn+".tex", out.Bytes(), os.ModePerm)
		}

		if tmpl == nil {
			log.Printf("template file is undefined, therefore, PDF will not be generated")
		} else if tmpl.Exec == "" {
			log.Printf("template file does not define the Exec entry, PDF will not be generated")
		} else {
			x := exec.Command("context", tmpl.Exec)
			x.Stderr = os.Stderr
			x.Stdout = os.Stdout
			x.Dir = workdir
			err = x.Run()
			if err != nil {
				log.Fatal(err)
			}

			if outFN != "" {
				genFN := strings.TrimSuffix(tmpl.Exec, filepath.Ext(tmpl.Exec)) + ".pdf"
				genFN = filepath.Join(workdir, genFN)
				log.Printf("generated pdf: %s\n", genFN)
				log.Printf("moving to: %s\n", outFN)
				err = os.Rename(genFN, outFN)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		log.Printf("mission accomplished\n")

	}

	app.Run(os.Args)
}
