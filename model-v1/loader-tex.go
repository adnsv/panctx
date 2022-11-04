package modelv1

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type projLoader struct {
	Title      string          `yaml:"title"`
	TopHeading string          `yaml:"top-heading"`
	Content    contentLoader   `yaml:"content"`
	Targets    []*targetLoader `yaml:"targets"`
}

type contentLoader struct {
	FrontMatter []*inputLoader `yaml:"frontmatter"`
	BodyMatter  []*inputLoader `yaml:"bodymatter"`
	Appendices  []*inputLoader `yaml:"appendices"`
	BackMatter  []*inputLoader `yaml:"backmatter"`
}

type inputLoader struct {
	File string `yaml:"file"`
	Tmpl string `yaml:"tmpl"`
}

type targetLoader struct {
	Name        string            `yaml:"name"`
	Group       string            `yaml:"group"`
	BasedOn     string            `yaml:"based-on"`
	Template    string            `yaml:"template"`
	Definitions map[string]string `yaml:"definitions"`
	Filter      *filterLoader     `yaml:"filter"`
	OutputFN    string            `yaml:"output-file"`
}

type filterLoader struct {
	Type string `yaml:"type"`
}

func newFileInput(fn string) (FileInput, error) {
	if fn == "" {
		return nil, errors.New("empty filename")
	}
	ext := strings.ToLower(filepath.Ext(fn))
	if ext == "" {
		return nil, fmt.Errorf("missing file extension in %s", fn)
	}
	if stat, err := os.Stat(fn); err != nil {
		return nil, fmt.Errorf("%w", err)
	} else if stat.IsDir() {
		return nil, fmt.Errorf("path %s points to a directory, not a file", fn)
	}

	switch ext {
	case ".tex":
		in := &TexInput{}
		in.fileName = fn
		return in, nil

	case ".svg":
	case ".png":
	case ".jpeg":
	case ".jpg":
		in := &ImageInput{}
		in.fileName = fn
		return in, nil

	default:
		in := &PandocInput{}
		in.fileName = fn
		return in, nil
	}
	return nil, fmt.Errorf("unsupported file extension in %s", fn)
}

func ProjectFromFile(yamlfn string) (*Proj, error) {
	log.Printf("loading project from %s\n", yamlfn)
	buf, err := os.ReadFile(yamlfn)
	if err != nil {
		return nil, err
	}
	return ProjectFromBytes(filepath.Dir(yamlfn), buf)
}

func ProjectFromBytes(projdir string, inputbuf []byte) (*Proj, error) {
	lp := projLoader{}
	err := yaml.Unmarshal(inputbuf, &lp)
	if err != nil {
		return nil, err
	}
	return projectFromLoader(projdir, &lp)
}

func projectFromLoader(projdir string, lp *projLoader) (*Proj, error) {
	prj := &Proj{}
	prj.Title = lp.Title
	prj.TopHeading = lp.TopHeading

	prj.fileInputs = map[string]FileInput{}
	prj.texInputs = map[string]*TexInput{}
	prj.pandocInputs = map[string]*PandocInput{}
	prj.imageInputs = map[string]*ImageInput{}

	normalizeFilename := func(fn string) string {
		if !filepath.IsAbs(fn) {
			fn = filepath.Join(projdir, fn)
		}
		return filepath.Clean(fn)
	}

	populateInputs := func(loaders []*inputLoader) ([]Input, error) {
		out := []Input{}

		for _, loader := range loaders {
			if loader.File != "" && loader.Tmpl != "" {
				return nil, errors.New("input must specify either a 'file' field or a 'tmpl' field, not both")
			}

			if loader.File != "" {
				fn := normalizeFilename(loader.File)
				in := prj.fileInputs[fn]
				if in == nil {
					var err error
					in, err = newFileInput(fn)
					if err != nil {
						return nil, err
					}
					prj.fileInputs[fn] = in
					switch in := in.(type) {
					case *TexInput:
						prj.texInputs[fn] = in
					case *PandocInput:
						prj.pandocInputs[fn] = in
					case *ImageInput:
						prj.imageInputs[fn] = in
					}
				}
				out = append(out, in)
			} else if loader.Tmpl != "" {
				out = append(out, &TemplateInput{TemplateName: loader.Tmpl})
			} else {
				return nil, errors.New("input must specify either a 'file' field or a 'tmpl' field")
			}
		}
		return out, nil
	}

	var err error
	prj.Content.FrontMatter, err = populateInputs(lp.Content.FrontMatter)
	if err != nil {
		return nil, err
	}
	prj.Content.BodyMatter, err = populateInputs(lp.Content.BodyMatter)
	if err != nil {
		return nil, err
	}
	prj.Content.Appendices, err = populateInputs(lp.Content.Appendices)
	if err != nil {
		return nil, err
	}
	prj.Content.BackMatter, err = populateInputs(lp.Content.BackMatter)
	if err != nil {
		return nil, err
	}

	return prj, nil
}
