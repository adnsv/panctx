package model

type Project struct {
	FrontMatter []string           `yaml:"frontmatter"`
	BodyMatter  []string           `yaml:"bodymatter"`
	Appendices  []string           `yaml:"appendices"`
	BackMatter  []string           `yaml:"backmatter"`
	Definitions map[string]string  `yaml:"definitions"`
	Targets     map[string]*Target `yaml:"target"`

	Layouts map[string]string

	// runtime helpers
	Main           *FileAsset
	TemplateAssets map[string]*FileAsset
	FileAssets     map[string]*FileAsset
}

type Target struct {
	Args        []string          `yaml:"args"`
	Output      string            `yaml:"output"`
	Definitions map[string]string `yaml:"definitions"`
}
