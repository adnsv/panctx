package model

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func (prj *Project) LoadTemplate(jsonFN string) (err error) {
	jsonFN, _ = filepath.Abs(jsonFN)
	jsonDIR := filepath.ToSlash(filepath.Dir(jsonFN))

	log.Printf("loading template config '%s'\n", jsonFN)
	buf, err := os.ReadFile(jsonFN)
	if err != nil {
		return err
	}

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
		t := assetTypeFromFileExt(filepath.Ext(v))
		a, err := prj.loadAsset(jsonDIR, t, v)
		if err != nil {
			return err
		}
		prj.TemplateAssets[v] = a
	}
	return
}
