package context

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/adnsv/go-utils/fs"
	"gopkg.in/yaml.v3"
)

type Template struct {
	path string `yaml:"-"`
	dir  string `yaml:"-"`

	Title                     string `yaml:"title"`
	Subtitle                  string `yaml:"subtitle"`
	Date                      string `yaml:"date"`
	DefaultExternalFigureSize string `yaml:"default-externalfigure-size"`

	FontSize  string            `yaml:"fontsize"`
	PageSize  string            `yaml:"pagesize"`
	PaperSize string            `yaml:"papersize"`
	Layout    string            `yaml:"layout"`
	Layouts   map[string]string `yaml:"layouts"`

	TopHeading string `yaml:"top-heading"` // part, chapter, or section (default)

	Exec string `yaml:"exec"`

	Files []*File `yaml:"files"`

	FrontMatter string `yaml:"front-matter"`
	BodyMatter  string `yaml:"body-matter"`
	Appendices  string `yaml:"appendices"`
	BackMatter  string `yaml:"back-matter"`
}

type File struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`

	absSrc string
	absDst string
}

type Document struct {
}

func NormalizePath(dir string, fn string) (string, error) {
	if fn == "" {
		return fn, nil
	}
	if !filepath.IsAbs(fn) {
		fn = filepath.Join(dir, fn)
	}
	fn, err := filepath.Abs(fn)
	if err != nil {
		return fn, err
	}
	return filepath.ToSlash(fn), nil
}

func OpenTemplate(fn string) (*Template, error) {

	buf, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	t := &Template{path: fn, dir: filepath.Dir(fn)}
	err = yaml.Unmarshal(buf, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Template) NormalizeSrcPath(fn string) (string, error) {
	return NormalizePath(t.dir, fn)
}

func (t *Template) CalcPaths(wd string) (err error) {
	for _, f := range t.Files {
		if f.Src == "" {
			continue
		}

		f.absSrc, err = t.NormalizeSrcPath(f.Src)
		if err != nil {
			return
		}

		fn := f.Dst
		if fn == "" {
			fn = f.Src
			if filepath.IsAbs(fn) {
				fn = filepath.Base(fn)
			}
		}
		f.absDst, err = NormalizePath(wd, fn)
		if err != nil {
			return
		}
	}
	return nil
}

func (t *Template) RelativeFilenameMapping(wd string) (map[string]string, error) {
	err := t.CalcPaths(wd)
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for _, f := range t.Files {
		if f.Src == "" {
			continue
		}
		ret[f.absSrc] = f.absDst
	}
	return ret, nil
}

func (t *Template) Execute(wd string, postProcess func(b []byte) []byte) (err error) {
	log.Printf("executing template\n")

	// choose Layout from Layouts[PageSize], if defined
	if lt, ok := t.Layouts[t.PageSize]; ok {
		t.Layout = lt
	}

	err = t.CalcPaths(wd)
	if err != nil {
		return err
	}
	for _, f := range t.Files {
		log.Printf("processing %s\n", f.Src)
		fin := f.Src
		if fin == "" {
			return errors.New("missing file src attribute")
		}
		if !path.IsAbs(fin) {
			fin, err = filepath.Abs(filepath.Join(t.dir, fin))
			if err != nil {
				return err
			}
		}
		gt, err := template.ParseFiles(fin)
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}

		err = gt.Execute(buf, t)
		if err != nil {
			return err
		}

		bb := buf.Bytes()
		if postProcess != nil {
			bb = postProcess(bb)
		}

		fout := f.Dst
		if fout == "" {
			fout = f.Src
		}
		fout = filepath.Join(wd, filepath.Base(fout))

		log.Printf("writing processed file to %s\n", fout)
		err = fs.WriteFileIfChanged(fout, bb)
		if err != nil {
			return err
		}
	}
	return nil
}
