package ki

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type EvtDetails struct {
	evt                 *fsnotify.Event
	isIgnored           bool
	isGo                bool
	isOther             bool
	isCriticalCSS       bool
	isNormalCSS         bool
	isWaveCSS           bool
	wfc                 *WatchedFile
	isNonEmptyCHMODOnly bool
	is_full_dev_reset   bool
}

func (c *Config) getEvtDetails(evt fsnotify.Event) *EvtDetails {
	if evt.Name == "" {
		return nil
	}

	cssImportURLsMu.RLock()
	_, isImportedCritical := criticalReliedUponFiles[evt.Name]
	_, isImportedNormal := normalReliedUponFiles[evt.Name]
	cssImportURLsMu.RUnlock()

	isCriticalCSS := evt.Name == c.cleanSources.CriticalCSSEntry || isImportedCritical
	isNormalCSS := evt.Name == c.cleanSources.NonCriticalCSSEntry || isImportedNormal

	isWaveCSS := isCriticalCSS || isNormalCSS

	var matchingWatchedFile *WatchedFile

	for _, wfc := range c._uc.Watch.Include {
		isMatch := c.get_is_match(potentialMatch{pattern: wfc.Pattern, path: evt.Name})
		if isMatch {
			matchingWatchedFile = &wfc
			break
		}
	}

	if matchingWatchedFile == nil {
		for _, wfc := range c.defaultWatchedFiles {
			isMatch := c.get_is_match(potentialMatch{pattern: wfc.Pattern, path: evt.Name})
			if isMatch {
				matchingWatchedFile = &wfc
				break
			}
		}
	}

	isGo := filepath.Ext(evt.Name) == ".go"
	if isGo && matchingWatchedFile != nil && matchingWatchedFile.TreatAsNonGo {
		isGo = false
	}

	isOther := !isGo && !isWaveCSS

	isIgnored := c.get_is_ignored(evt.Name, c.ignoredFilePatterns)
	if isOther && matchingWatchedFile == nil {
		isIgnored = true
	}

	var is_full_dev_reset bool
	if matchingWatchedFile != nil {
		for _, hook := range matchingWatchedFile.OnChangeHooks {
			if hook.Cmd == __internal_full_dev_reset_less_go_mrkr {
				is_full_dev_reset = true
				break
			}
		}
	}

	return &EvtDetails{
		evt:                 &evt,
		isOther:             isOther,
		isWaveCSS:           isWaveCSS,
		isGo:                isGo,
		isIgnored:           isIgnored,
		isCriticalCSS:       isCriticalCSS,
		isNormalCSS:         isNormalCSS,
		wfc:                 matchingWatchedFile,
		isNonEmptyCHMODOnly: c.getIsNonEmptyCHMODOnly(evt),
		is_full_dev_reset:   is_full_dev_reset,
	}
}

func (c *Config) getIsEmptyFile(evt fsnotify.Event) bool {
	file, err := os.Open(evt.Name)
	if err != nil {
		c.Logger.Error(fmt.Sprintf("error: failed to open file: %v", err))
		return false
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		c.Logger.Error(fmt.Sprintf("error: failed to get file stats: %v", err))
		return false
	}
	return stat.Size() == 0
}

func (c *Config) getIsNonEmptyCHMODOnly(evt fsnotify.Event) bool {
	isSolelyCHMOD := !evt.Has(fsnotify.Write) && !evt.Has(fsnotify.Create) && !evt.Has(fsnotify.Remove) && !evt.Has(fsnotify.Rename)
	return isSolelyCHMOD && !c.getIsEmptyFile(evt)
}
