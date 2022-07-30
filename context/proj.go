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

type Project struct {
	MainDir   string
	ConfigDir string
	WorkDir   string

	Definitions    map[string]string
	Layouts        map[string]string
	MarkdownAssets []*MarkdownAsset
	TemplateAssets []*TemplateAsset

	mainBuf   []byte
	mainDstFN string
}

type MarkdownAsset struct {
	srcFN string
	dstFN string
	jbuf  []byte
	d     *pandoc.Document
}

type TemplateAsset struct {
	srcFN string
	dstFN string
}

func NewProject(workdir string) *Project {
	return &Project{
		WorkDir:     workdir,
		Definitions: map[string]string{},
		Layouts:     map[string]string{},
	}
}

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
