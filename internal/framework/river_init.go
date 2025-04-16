package framework

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"path"

	"github.com/river-now/river/kit/mux"
)

func (h *River) Init(isDev bool) {
	if err := h.initInner(isDev); err != nil {
		errMsg := fmt.Sprintf("Error initializing River: %v", err)
		if isDev {
			Log.Error(errMsg)
		} else {
			panic(errMsg)
		}
	} else {
		Log.Info("River initialized", "build id", h._buildID)
	}
}

// RUNTIME! Gets called from the handler maker, which gets called by the user's router init function.
func (h *River) validateAndDecorateNestedRouter(nestedRouter *mux.NestedRouter) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if nestedRouter == nil {
		panic("nestedRouter is nil")
	}
	for _, p := range h._paths {
		_is_already_registered := nestedRouter.IsRegistered(p.Pattern)
		if !_is_already_registered {
			mux.RegisterNestedPatternWithoutHandler(nestedRouter, p.Pattern)
		}
	}
	for pattern := range nestedRouter.AllRoutes() {
		if _, exists := h._paths[pattern]; !exists {
			Log.Error(fmt.Sprintf("Warning: no client-side route found for pattern %v.", pattern))
		}
	}
}

func PrettyPrintFS(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			fmt.Println(path)
		} else {
			fmt.Printf("%s (%s)\n", path, d.Type())
		}
		return nil
	})
}

func (h *River) initInner(isDev bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h._isDev = isDev
	privateFS, err := h.Kiruna.GetPrivateFS()
	if err != nil {
		errMsg := fmt.Sprintf("could not get private fs: %v", err)
		Log.Error(errMsg)
		return errors.New(errMsg)
	}
	h._privateFS = privateFS
	pathsFile, err := h.getBasePaths_StageOneOrTwo(isDev)
	if err != nil {
		errMsg := fmt.Sprintf("could not get base paths: %v", err)
		Log.Error(errMsg)
		return errors.New(errMsg)
	}
	h._buildID = pathsFile.BuildID
	if h._paths == nil {
		h._paths = make(map[string]*Path, len(pathsFile.Paths))
	}
	for _, p := range pathsFile.Paths {
		h._paths[p.Pattern] = p
	}
	h._clientEntrySrc = pathsFile.ClientEntrySrc
	h._clientEntryOut = pathsFile.ClientEntryOut
	h._clientEntryDeps = pathsFile.ClientEntryDeps
	h._depToCSSBundleMap = pathsFile.DepToCSSBundleMap
	if h._depToCSSBundleMap == nil {
		h._depToCSSBundleMap = make(map[string]string)
	}
	tmpl, err := template.ParseFS(h._privateFS, h.Kiruna.GetRiverHTMLTemplateLocation())
	if err != nil {
		return fmt.Errorf("error parsing root template: %v", err)
	}
	h._rootTemplate = tmpl
	return nil
}

func (h *River) getBasePaths_StageOneOrTwo(isDev bool) (*PathsFile, error) {
	fileToUse := RiverPathsStageOneJSONFileName
	if !isDev {
		fileToUse = RiverPathsStageTwoJSONFileName
	}
	pathsFile := PathsFile{}
	file, err := h._privateFS.Open(path.Join("river_out", fileToUse))
	if err != nil {
		errMsg := fmt.Sprintf("could not open %s: %v", fileToUse, err)
		Log.Error(errMsg)
		return nil, errors.New(errMsg)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&pathsFile)
	if err != nil {
		errMsg := fmt.Sprintf("could not decode %s: %v", fileToUse, err)
		Log.Error(errMsg)
		return nil, errors.New(errMsg)
	}
	return &pathsFile, nil
}
