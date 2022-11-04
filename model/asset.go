package model

import (
	"github.com/adnsv/go-pandoc"
)

type AssetType int

const (
	AssetUnsupported = AssetType(iota)
	AssetIgnore
	AssetCopy // just transfer to workdir
	AssetScanReplace
	AssetPandoc
	AssetSVG
)

type FileAsset struct {
	AbsSrcFilePath  string
	Type            AssetType
	RelWorkFilePath string
	SrcContent      []byte
	PandocDOM       *pandoc.Document
}

func (t *AssetType) String() string {
	switch *t {
	case AssetUnsupported:
		return "unsupported"
	case AssetIgnore:
		return "ignored"
	case AssetCopy:
		return "copy"
	case AssetScanReplace:
		return "scan-replace"
	case AssetPandoc:
		return "pandoc-converted"
	case AssetSVG:
		return "svg"
	default:
		return "<invalid>"
	}
}
