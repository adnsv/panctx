package modelv1

type Proj struct {
	Title      string
	TopHeading string
	Content    Content
	Targets    []*Target

	fileInputs   map[string]FileInput
	texInputs    map[string]*TexInput
	pandocInputs map[string]*PandocInput
	imageInputs  map[string]*ImageInput
}

type Content struct {
	FrontMatter []Input
	BodyMatter  []Input
	Appendices  []Input
	BackMatter  []Input
}

type Input interface {
}

type ContentInput interface {
}

type TemplateInput struct {
	TemplateName string
}

type FileInput interface {
	FileName() string
}

type baseFileInput struct {
	fileName string // absolute
}

func (fi *baseFileInput) FileName() string {
	return fi.fileName
}

type TexInput struct {
	baseFileInput
}

type PandocInput struct {
	baseFileInput
}

type ImageInput struct {
	baseFileInput
}

type Target struct {
	Name        string
	BasedOn     string `yaml:"based-on"`
	Template    string
	Definitions map[string]string
}
