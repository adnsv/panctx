package model

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adnsv/go-pandoc"
	"github.com/adnsv/go-utils/filesystem"
)

func (prj *Project) loadAsset(basedir string, t AssetType, fn string) (a *FileAsset, err error) {

	// todo: track origin and circular references
	absFN := normalizePath(basedir, fn)
	if a = prj.FileAssets[absFN]; a != nil {
		if a.Type != t {
			log.Printf("failed to load template asset '%s' as %s\n", fn, t.String())
			log.Printf("it was prefiously imported as '%s'\b", a.Type.String())

			return nil, errors.New("failed to load template asset")
		}
		return a, nil
	}

	log.Printf("loading asset '%s'\n", absFN)

	if err := filesystem.ValidateFileExists(absFN); err != nil {
		return nil, err
	}

	a = &FileAsset{
		AbsSrcFilePath: absFN,
		Type:           t,
	}
	prj.FileAssets[absFN] = a

	absDIR := filepath.Dir(absFN)

	switch a.Type {
	case AssetScanReplace:
		a.SrcContent, err = os.ReadFile(absFN)
		if err != nil {
			return nil, err
		}
		err = prj.scanWildcards(absDIR, a.SrcContent)
		if err != nil {
			return nil, err
		}
	case AssetPandoc:
		log.Printf("running pandoc -t json %s\n", absFN)
		jbuf, err := exec.Command("pandoc", "-t", "json", absFN).Output()
		if err != nil {
			return nil, fmt.Errorf("pandoc error: %w", err)
		}
		a.PandocDOM, err = pandoc.NewDocument(jbuf)
		if err != nil {
			return nil, err
		}
		for k, v := range a.PandocDOM.ParseMeta() {
			prj.Definitions[k] = v
		}
	case AssetSVG:
		// will load and normalize later
	}
	return a, nil
}

func assetTypeFromFileExt(ext string) AssetType {
	switch strings.ToLower(ext) {
	case ".md":
	case ".markdown":
		return AssetPandoc
	case ".tex":
		return AssetScanReplace
	case ".svg":
		return AssetSVG
	case ".bmp":
	case ".jpeg":
	case ".png":
		return AssetCopy
	}
	return AssetUnsupported
}

func normalizePath(refdir string, fn string) string {
	if fn == "" {
		return fn
	}
	if !filepath.IsAbs(fn) {
		fn = filepath.Join(refdir, fn)
	}
	fn = filepath.Clean(fn)
	return filepath.ToSlash(fn)
}
