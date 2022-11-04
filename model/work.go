package model

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adnsv/go-utils/filesystem"
	"github.com/adnsv/panctx/context"
)

func (prj *Project) ChooseWorkFNs(cb func(a *FileAsset) string) error {
	m := map[string]struct{}{}

	exists := func(workFN string) bool {
		_, ok := m[workFN]
		return ok
	}

	if prj.Main != nil {
		if prj.Main.RelWorkFilePath == "" {
			prj.Main.RelWorkFilePath = cb(prj.Main)
		}
		m[prj.Main.RelWorkFilePath] = struct{}{}
	}
	for _, a := range prj.FileAssets {
		if a.RelWorkFilePath == "" {
			fn := cb(a)
			if exists(fn) {
				ext := filepath.Ext(fn)
				base := ext[:len(fn)-len(ext)]
				n := 2
				fn = fmt.Sprintf("%s%d.%s", base, n, ext)
				for exists(fn) {
					n++
					fn = fmt.Sprintf("%s%d.%s", base, n, ext)
				}
			}
			a.RelWorkFilePath = fn
			m[prj.Main.RelWorkFilePath] = struct{}{}
		}
		m[a.RelWorkFilePath] = struct{}{}
	}
	return nil
}

func (prj *Project) processAsset(a *FileAsset) error {
	if a.RelWorkFilePath == "" {
		return nil
	}

	targetFN := filepath.Join(prj.WorkDir, a.RelWorkFilePath)

	switch a.Type {
	case AssetIgnore:
		log.Printf("ignoring %s\n", a.AbsSrcFilePath)
		return nil

	case AssetCopy:
		log.Printf("copying %s -> %s\n", a.AbsSrcFilePath, a.RelWorkFilePath)
		if _, e := os.Stat(targetFN); e == nil {
			os.Remove(targetFN)
		}
		err := os.Link(a.AbsSrcFilePath, targetFN)
		return err

	case AssetScanReplace:
		log.Printf("converting %s -> %s\n", a.AbsSrcFilePath, a.RelWorkFilePath)
		baseDIR := filepath.Dir(a.AbsSrcFilePath)
		out, _ := ReplaceWildcards(a.SrcContent, func(k, v string) (string, error) {
			if k == "var" {
				r, ok := prj.Definitions[v]
				if ok {
					return EscapeStr(r), nil
				} else {
					return "", fmt.Errorf("unknown variable: %s", v)
				}
			} else if k == "template" {
				from_asset := prj.templateAssets[v]
				if from_asset == nil {
					return "<MISSING>", fmt.Errorf("cannot resolve reference '%s:%s'", k, v)
				}
				if from_asset.RelWorkFilePath == "" {
					return "<MISSING>", fmt.Errorf("reference '%s:%s' does not resolve to a temporary", k, v)
				}
				return from_asset.RelWorkFilePath, nil
			} else {
				// handle as file
				from_fn := normalizePath(baseDIR, v)
				from_asset := prj.fileAssets[from_fn]

				if from_asset == nil {
					return "<MISSING>", fmt.Errorf("cannot resolve reference '%s:%s'", k, v)
				}
				if from_asset.RelWorkFilePath == "" {
					return "<MISSING>", fmt.Errorf("reference '%s:%s' does not resolve to a temporary", k, v)
				}
				return from_asset.RelWorkFilePath, nil
			}
		})
		err := filesystem.WriteFileIfChanged(targetFN, out)
		return err

	case AssetPandoc:
		out := bytes.Buffer{}
		srcDIR := filepath.Dir(a.AbsSrcFilePath)
		w := context.NewWriter(&out)
		w.OnResolveImageTarget = func(url string) string {
			fn := normalizePath(srcDIR, url)
			if t := prj.FileAssets[fn]; t != nil {
				if t.RelWorkFilePath != "" {
					return t.RelWorkFilePath
				}
			}
			return fn
		}

		w.SetTopLevelDivision(prj.Definitions["top-heading"])
		w.DefaultExternalFigureSize = prj.Definitions["default-externalfigure-size"]
		flow, err := a.PandocDOM.Flow()
		if err != nil {
			return err
		}
		w.WriteBlocks(flow)
		log.Printf("- writing %s\n", targetFN)
		err = filesystem.WriteFileIfChanged(targetFN, out.Bytes())
		if err != nil {
			return err
		}

	case AssetSVG:
		// todo: reprocess content
		log.Printf("copying %s -> %s\n", a.AbsSrcFilePath, a.RelWorkFilePath)
		if _, e := os.Stat(targetFN); e == nil {
			os.Remove(targetFN)
		}
		err := os.Link(a.AbsSrcFilePath, targetFN)
		return err
	}
	return nil
}

func (prj *Project) Process() (err error) {
	err = prj.AssignWorkFNs()
	if err != nil {
		return err
	}

	for _, a := range prj.FileAssets {
		err = prj.processAsset(a)
		if err != nil {
			return err
		}
	}

	return nil
}
