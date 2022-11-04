package context

import (
	"path/filepath"

	"github.com/adnsv/panctx/model"
)

type Filter struct {
}

func (f *Filter) ChooseOutputName(a *model.FileAsset) string {
	b := filepath.Base(a.AbsSrcFilePath)
	if a.Type == model.AssetPandoc {
		return replaceEXT(b, ".tex")
	} else {
		return b
	}
}

func (f *Filter) ProcessAsset(a *model.FileAsset, workDIR string) error {

}

func replaceEXT(fn, newEXT string) string {
	return fn[:len(fn)-len(filepath.Ext(fn))] + newEXT
}
