package model

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func LoadMainYaml(mainFN string) (*Project, error) {

	mainFN, err := filepath.Abs(mainFN)
	if err != nil {
		return nil, err
	}

	feedback := func(format string, args ...any) {
		fmt.Printf(format, args...)
	}

	feedback("loading project from %s\n", mainFN)
	buf, err := os.ReadFile(mainFN)
	if err != nil {
		return nil, err
	}

	prj := Project{}
	err = yaml.Unmarshal(buf, &prj)
	return &prj, err
}
