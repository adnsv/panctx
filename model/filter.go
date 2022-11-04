package model

type Filter interface {
	ChooseOutputName(a *FileAsset) string
	ProcessAsset(a *FileAsset, workDIR string) error
}
