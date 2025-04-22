package framework

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"path"

	"github.com/river-now/river/kit/mux"
)

func (h *River) Init(isDev bool) {
	if err := h.initInner(isDev); err != nil {
		wrapped := fmt.Errorf("error initializing River: %w", err)
		if isDev {
			Log.Error(wrapped.Error())
		} else {
			panic(wrapped)
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
	privateFS, err := h.Wave.GetPrivateFS()
	if err != nil {
		wrapped := fmt.Errorf("could not get private fs: %w", err)
		Log.Error(wrapped.Error())
		return wrapped
	}
	h._privateFS = privateFS
	pathsFile, err := h.getBasePaths_StageOneOrTwo(isDev)
	if err != nil {
		wrapped := fmt.Errorf("could not get base paths: %w", err)
		Log.Error(wrapped.Error())
		return wrapped
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
	tmpl, err := template.ParseFS(h._privateFS, h.Wave.GetRiverHTMLTemplateLocation())
	if err != nil {
		return fmt.Errorf("error parsing root template: %w", err)
	}
	h._rootTemplate = tmpl
	headElsInstance.InitUniqueRules(h.GetHeadElUniqueRules())
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
		wrapped := fmt.Errorf("could not open %s: %v", fileToUse, err)
		Log.Error(wrapped.Error())
		return nil, wrapped
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&pathsFile)
	if err != nil {
		wrapped := fmt.Errorf("could not decode %s: %v", fileToUse, err)
		Log.Error(wrapped.Error())
		return nil, wrapped
	}
	return &pathsFile, nil
}
