// Package context provides the core conversion logic for panctx, handling template
// configuration, Markdown to ConTeXt conversion, and PDF generation.
package context

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adnsv/go-pandoc"
	"github.com/adnsv/go-utils/filesystem"
	"gopkg.in/yaml.v3"
)

// Project represents a document conversion project, managing template configuration,
// asset processing, and PDF generation. It coordinates the conversion pipeline from
// Markdown through ConTeXt to PDF.
type Project struct {
	MainDir   string // Directory containing the main input file
	ConfigDir string // Directory containing the template configuration
	WorkDir   string // Working directory for intermediate files

	Definitions    map[string]string // Variable definitions for template substitution
	Layouts        map[string]string // Page layout configurations mapped by page size
	MarkdownAssets []*MarkdownAsset  // Markdown files to be converted
	TemplateAssets []*TemplateAsset  // Template assets to be processed

	mainBuf   []byte // Buffer containing the main input file content
	mainDstFN string // Destination path for the processed main file
}

// MarkdownAsset represents a Markdown file asset that will be converted to ConTeXt.
// It tracks the source file, destination file, Pandoc JSON buffer, and parsed document.
type MarkdownAsset struct {
	srcFN string            // Source Markdown file path
	dstFN string            // Destination ConTeXt file path
	jbuf  []byte            // Pandoc JSON output buffer
	d     *pandoc.Document  // Parsed Pandoc document
}

// TemplateAsset represents a template file asset that will be processed and copied
// to the working directory with variable substitution applied.
type TemplateAsset struct {
	srcFN string // Source template file path
	dstFN string // Destination file path in working directory
}

// NewProject creates a new Project instance with the specified working directory.
// It initializes empty maps for Definitions and Layouts.
func NewProject(workdir string) *Project {
	return &Project{
		WorkDir:     workdir,
		Definitions: map[string]string{},
		Layouts:     map[string]string{},
	}
}

// LoadConfig loads the template configuration from a YAML file. It parses variable
// definitions, page layouts, and asset paths. Template assets are validated and
// registered for processing.
func (prj *Project) LoadConfig(fn string) (err error) {
	log.Printf("loading template config from %s\n", fn)
	buf, err := os.ReadFile(fn)
	if err != nil {
		return err
	}

	prj.ConfigDir = filepath.Dir(fn)

	type templateLoader struct {
		Definitions map[string]string `yaml:"def"`
		Layouts     map[string]string `yaml:"layouts"`
		Assets      []string          `yaml:"assets"`
	}

	t := templateLoader{}
	err = yaml.Unmarshal(buf, &t)
	if err != nil {
		return err
	}

	for k, v := range t.Definitions {
		prj.Definitions[k] = v
	}

	for k, v := range t.Layouts {
		prj.Layouts[k] = v
	}

	for _, v := range t.Assets {
		log.Printf("- loading asset %s\n", v)
		a := &TemplateAsset{}
		a.srcFN, err = normalizePath(prj.ConfigDir, v)
		if err != nil {
			return err
		}
		stat, err := os.Stat(a.srcFN)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return fmt.Errorf("path '%s' points to a directory instead of a file", a.srcFN)
		}

		a.dstFN = filepath.ToSlash(filepath.Join(prj.WorkDir, filepath.Base(a.srcFN)))
		prj.TemplateAssets = append(prj.TemplateAssets, a)
	}
	return
}

var re = regexp.MustCompile(`(?m)\$<((?:[^>\$])*)>\$`)

// LoadMain loads the main input file and scans it for Markdown asset references.
// For each Markdown asset found, it converts it to Pandoc JSON format and extracts
// metadata into the project's Definitions map.
func (prj *Project) LoadMain(fn string) (err error) {
	log.Printf("loading main from %s\n", fn)
	prj.mainBuf, err = os.ReadFile(fn)
	if err != nil {
		return
	}
	prj.MainDir, err = filepath.Abs(filepath.Dir(fn))
	prj.mainDstFN = filepath.ToSlash(filepath.Join(prj.WorkDir, filepath.Base(fn)))

	for _, s := range re.FindAllString(string(prj.mainBuf), -1) {
		s = strings.TrimPrefix(s, "$<")
		s = strings.TrimSuffix(s, ">$")
		i := strings.IndexByte(s, ':')
		if i < 0 {
			continue
		}
		if s[:i] == "markdown" {
			fn := s[i+1:]
			log.Printf("found markdown asset: %s\n", fn)
			fn, err = normalizePath(prj.MainDir, fn)
			if err != nil {
				return
			}
			stat, err := os.Stat(fn)
			if err != nil {
				return err
			}
			if stat.IsDir() {
				return fmt.Errorf("path '%s' points to a directory instead of a file", fn)
			}
			md := &MarkdownAsset{srcFN: fn}
			md.dstFN = filepath.ToSlash(filepath.Join(prj.WorkDir, filepath.Base(fn)+".tex"))

			md.jbuf, err = exec.Command("pandoc", "-t", "json", fn).Output()
			if err != nil {
				return fmt.Errorf("pandoc error: %w", err)
			}
			md.d, err = pandoc.NewDocument(md.jbuf)
			if err != nil {
				return err
			}
			for k, v := range md.d.ParseMeta() {
				prj.Definitions[k] = v
			}

			prj.MarkdownAssets = append(prj.MarkdownAssets, md)
		}
	}

	return
}

// replaceContent performs variable and asset path substitution on the given buffer.
// It replaces $<var:name>$, $<template:path>$, and $<markdown:path>$ placeholders
// with their corresponding values or file paths.
func (prj *Project) replaceContent(buf []byte) []byte {
	return re.ReplaceAllFunc(buf, func(v []byte) []byte {
		s := string(v)
		s = strings.TrimPrefix(s, "$<")
		s = strings.TrimSuffix(s, ">$")
		i := strings.IndexByte(s, ':')
		if i > 0 {
			k := s[:i]
			v := s[i+1:]
			switch k {
			case "var":
				r, ok := prj.Definitions[v]
				if ok {
					return []byte(EscapeStr(r))
				} else {
					log.Printf("unknown variable: %s\n", v)
				}
			case "template":
				fn, err := normalizePath(prj.ConfigDir, v)
				if err != nil {
					log.Printf("invalid template path: %s\n", v)
					break
				}
				for _, tf := range prj.TemplateAssets {
					if tf.srcFN == fn {
						return []byte(tf.dstFN)
					}
				}
				log.Printf("unknown template path: %s\n", v)

			case "markdown":
				fn, err := normalizePath(prj.MainDir, v)
				if err != nil {
					log.Printf("invalid template path: %s\n", v)
					break
				}
				for _, tf := range prj.MarkdownAssets {
					if tf.srcFN == fn {
						return []byte(tf.dstFN)
					}
				}
				log.Printf("unknown markdown path: %s\n", v)
			}

		}
		return v
	})
}

// Process converts all assets and generates ConTeXt output files. It processes the main
// file, template assets, and Markdown assets, writing the results to the working directory.
// Markdown assets are converted from Pandoc AST to ConTeXt format.
func (prj *Project) Process() (err error) {
	log.Printf("processing main file")

	out := prj.replaceContent(prj.mainBuf)
	log.Printf("- writing %s\n", prj.mainDstFN)
	err = filesystem.WriteFileIfChanged(prj.mainDstFN, out)
	if err != nil {
		return err
	}

	for _, f := range prj.TemplateAssets {
		log.Printf("processing %s\n", f.srcFN)
		in, err := os.ReadFile(f.srcFN)
		if err != nil {
			return err
		}
		out := prj.replaceContent(in)
		log.Printf("- writing %s\n", f.dstFN)
		err = filesystem.WriteFileIfChanged(f.dstFN, out)
		if err != nil {
			return err
		}
	}

	for _, f := range prj.MarkdownAssets {
		log.Printf("processing %s\n", f.srcFN)
		out := bytes.Buffer{}
		w := NewWriter(&out, filepath.Dir(f.srcFN))
		w.SetTopLevelDivision(prj.Definitions["top-heading"])
		w.DefaultExternalFigureSize = prj.Definitions["default-externalfigure-size"]
		flow, err := f.d.Flow()
		if err != nil {
			return err
		}
		w.WriteBlocks(flow)
		log.Printf("- writing %s\n", f.dstFN)
		err = filesystem.WriteFileIfChanged(f.dstFN, out.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

// BuildPDF generates a PDF file from the processed ConTeXt files by executing the
// context command. It returns the path to the generated PDF file or an error if
// the conversion fails.
func (prj *Project) BuildPDF() (pdf string, err error) {
	pdf = strings.TrimSuffix(prj.mainDstFN, filepath.Ext(prj.mainDstFN)) + ".pdf"
	log.Printf("generating PDF -> %s\n", pdf)

	x := exec.Command("context", prj.mainDstFN)
	x.Stderr = os.Stderr
	x.Stdout = os.Stdout
	x.Dir = prj.WorkDir
	err = x.Run()
	if err != nil {
		return "", fmt.Errorf("ConTEXt error: %w", err)
	}

	_, err = os.Stat(pdf)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("missing ConTEXt output: %w", err)
	} else if err != nil {
		return "", fmt.Errorf("missing ConTEXt output: %w", err)
	}
	return
}

// normalizePath converts a file path to an absolute path using forward slashes.
// If the path is relative, it is resolved relative to refdir. The resulting path
// is converted to use forward slashes for cross-platform compatibility.
func normalizePath(refdir string, fn string) (string, error) {
	if fn == "" {
		return fn, nil
	}
	if !filepath.IsAbs(fn) {
		fn = filepath.Join(refdir, fn)
	}
	fn, err := filepath.Abs(fn)
	if err != nil {
		return fn, err
	}
	return filepath.ToSlash(fn), nil
}
