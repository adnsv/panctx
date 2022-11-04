package model

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

func (prj *Project) scanWildcards(basedir string, contentBuf []byte) error {
	return ForAllWildcards(contentBuf, func(k, v string) error {
		if k == "var" {
			return nil
		}

		t := AssetUnsupported
		if k == "pandoc" || k == "md" || k == "markdown" {
			t = AssetPandoc
		} else if k == "context" || k == "tex" {
			t = AssetScanReplace
		} else if k == "svg" {
			t = AssetSVG
		} else if k == "bitmap" || k == "jpeg" || k == "png" {
			t = AssetCopy
		} else if k == "binary" {
			t = AssetCopy
		} else if k == "" || k == "auto" {
			t = assetTypeFromFileExt(filepath.Ext(v))
		} else if k == "template" {
			a := prj.TemplateAssets[v]
			if a == nil {
				return fmt.Errorf("unknown reference to 'template:%s'", v)
			} else {
				return nil
			}
		}

		if t == AssetUnsupported {
			log.Printf("unsupported asset type: %s.%s", k, v)
			return nil
		}

		_, err := prj.loadAsset(basedir, t, v)
		return err
	})
}

func (prj *Project) LoadMainTex(mainFN string) (err error) {

	mainFN, _ = filepath.Abs(mainFN)
	mainEXT := filepath.Ext(mainFN)
	mainDIR := filepath.ToSlash(filepath.Dir(mainFN))

	log.Printf("loading main file from %s\n", mainFN)

	if prj.Main != nil {
		err = fmt.Errorf("main file already loaded from %s", prj.Main.AbsSrcFilePath)
		return err
	}

	t := AssetUnsupported

	switch strings.ToLower(mainEXT) {
	case ".tex":
		t = AssetScanReplace
	default:
		return fmt.Errorf("unsupported main file type")
	}

	prj.Main, err = prj.loadAsset(mainDIR, t, mainFN)
	if err != nil {
		return err
	}

	return
}
