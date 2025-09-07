package ki

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sync/errgroup"
)

func (c *Config) process_batched_events(events []fsnotify.Event) {
	fileChanges := make(map[string]fsnotify.Event)
	for _, evt := range events {
		fileChanges[evt.Name] = evt
	}

	relevantFileChanges := make(map[string]*EvtDetails)

	wfcsAlreadyHandled := make(map[string]bool)
	isGoOrNeedsHardReloadEvenIfNonGo := false

	for _, evt := range fileChanges {
		configFilePath := c.GetConfigFile()
		if configFilePath != "" {
			isConfig := filepath.Clean(filepath.Join(c.cleanWatchRoot, evt.Name)) == filepath.Clean(configFilePath)
			isWriteOrCreate := evt.Has(fsnotify.Write) || evt.Has(fsnotify.Create)
			if isConfig && isWriteOrCreate {
				c.Logger.Info("[watcher]", "op", evt.Op.String(), "filename", evt.Name)
				c.MustStartDev(must_start_dev_opts{is_rebuild: true})
				return
			}
		}

		// no need to check error, because we want to process either way
		file_info_maybe_nil, _ := os.Stat(evt.Name)
		is_dir := file_info_maybe_nil != nil && file_info_maybe_nil.IsDir()

		if is_dir {
			if evt.Has(fsnotify.Create) || evt.Has(fsnotify.Rename) {
				if err := c.add_directory_to_watcher(evt.Name); err != nil {
					c.Logger.Error(fmt.Sprintf("error: failed to add directory to watcher: %v", err))
					continue
				}
			}
			continue
		}

		evtDetails := c.getEvtDetails(evt)
		if evtDetails == nil || evtDetails.isIgnored {
			continue
		}

		wfc := evtDetails.wfc
		if wfc == nil {
			wfc = &WatchedFile{}
		}

		if _, alreadyHandled := wfcsAlreadyHandled[wfc.Pattern]; alreadyHandled {
			continue
		}

		wfcsAlreadyHandled[wfc.Pattern] = true

		if !isGoOrNeedsHardReloadEvenIfNonGo {
			isGoOrNeedsHardReloadEvenIfNonGo = evtDetails.isGo
		}
		if !isGoOrNeedsHardReloadEvenIfNonGo {
			isGoOrNeedsHardReloadEvenIfNonGo = getNeedsHardReloadEvenIfNonGo(wfc)
		}

		relevantFileChanges[evt.Name] = evtDetails
	}

	if len(relevantFileChanges) == 0 {
		return
	}

	hasMultipleEvents := len(relevantFileChanges) > 1

	if !hasMultipleEvents {
		var evtName string
		for evtName = range relevantFileChanges {
			break
		}
		if relevantFileChanges[evtName].isNonEmptyCHMODOnly {
			return
		}
	}

	if hasMultipleEvents {
		allEvtsAreNonEmptyCHMODOnly := true

		for _, evtDetails := range relevantFileChanges {
			if evtDetails.isNonEmptyCHMODOnly {
				continue
			} else {
				allEvtsAreNonEmptyCHMODOnly = false
				break
			}
		}

		if allEvtsAreNonEmptyCHMODOnly {
			return
		}

		c.browserTabManager.broadcast <- refreshFilePayload{
			ChangeType: changeTypeRebuilding,
		}
	}

	eg := errgroup.Group{}
	if hasMultipleEvents && isGoOrNeedsHardReloadEvenIfNonGo {
		eg.Go(func() error {
			c.Logger.Info("Shutting down running app")
			c.kill_running_go_binary()
			return nil
		})
	}

	for _, evtDetails := range relevantFileChanges {
		c.Logger.Info("[watcher]", "op", evtDetails.evt.Op.String(), "filename", evtDetails.evt.Name)

		if evtDetails.is_full_dev_reset {
			c.MustStartDev(must_start_dev_opts{is_rebuild: true})
			return
		}

		err := c.mustHandleFileChange(evtDetails, hasMultipleEvents)
		if err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to handle file change: %v", err))
			return
		}
	}

	if hasMultipleEvents && isGoOrNeedsHardReloadEvenIfNonGo {
		if err := eg.Wait(); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to kill app: %v", err))
			return
		}
		c.Logger.Info("Restarting app")
		c.run_go_binary()
	}

	if hasMultipleEvents {
		c.must_reload_broadcast(
			refreshFilePayload{ChangeType: changeTypeOther},
			must_reload_broadcast_opts{
				wait_for_app:  true,
				wait_for_vite: c.isUsingVite(),
				message:       "Hard reloading browser",
			},
		)
	}
}

func getNeedsHardReloadEvenIfNonGo(wfc *WatchedFile) bool {
	return wfc.RecompileGoBinary || wfc.RestartApp
}

func (c *Config) mustHandleFileChange(
	evtDetails *EvtDetails,
	isPartOfBatch bool,
) error {
	wfc := evtDetails.wfc
	if wfc == nil {
		wfc = &WatchedFile{}
	}

	if c.is_using_browser() && !wfc.SkipRebuildingNotification && !evtDetails.isWaveCSS && !isPartOfBatch {
		c.browserTabManager.broadcast <- refreshFilePayload{
			ChangeType: changeTypeRebuilding,
		}
	}

	needsHardReloadEvenIfNonGo := getNeedsHardReloadEvenIfNonGo(wfc)

	needsKillAndRestart := (evtDetails.isGo || needsHardReloadEvenIfNonGo) && !isPartOfBatch

	killAndRestartEG := errgroup.Group{}
	if needsKillAndRestart {
		killAndRestartEG.Go(func() error {
			c.Logger.Info("Terminating running app")
			c.kill_running_go_binary()
			return nil
		})
	}

	sortedOnChanges := sortOnChangeCallbacks(wfc.OnChangeHooks)

	if sortedOnChanges.exists {
		// Wave has no control over error handling for "no-wait" callbacks.
		// They might not even be finished until after Wave has already
		// restarted the app (in fact, that's the point).
		go func() {
			_ = c.runConcurrentOnChangeCallbacks(sortedOnChanges.stratConcurrentNoWait, evtDetails.evt.Name, false)
		}()

		if err := c.simpleRunOnChangeCallbacks(sortedOnChanges.stratPre, evtDetails.evt.Name); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to build: %v", err))
			return err
		}

		if wfc.RunOnChangeOnly {
			c.Logger.Info("ran applicable onChange callbacks")
			return nil
		}

		eg := errgroup.Group{}
		//////////////// MAIN CALLBACK //////////////////
		eg.Go(func() error {
			return c.callback(wfc, evtDetails)
		})

		if err := c.runConcurrentOnChangeCallbacks(sortedOnChanges.stratConcurrent, evtDetails.evt.Name, true); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to build: %v", err))
			return err
		}

		if err := eg.Wait(); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to build: %v", err))
			return err
		}
	}

	if !sortedOnChanges.exists {
		if err := c.callback(wfc, evtDetails); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to build: %v", err))
			return err
		}
	}

	if err := c.simpleRunOnChangeCallbacks(sortedOnChanges.stratPost, evtDetails.evt.Name); err != nil {
		c.Logger.Error(fmt.Sprintf("error: failed to build: %v", err))
		return err
	}

	if needsKillAndRestart {
		if err := killAndRestartEG.Wait(); err != nil {
			c.Logger.Error(fmt.Sprintf("error: failed to kill app: %v", err))
			return err
		}
		c.Logger.Info("Restarting app")
		c.run_go_binary()
	}

	if !c.is_using_browser() || isPartOfBatch {
		return nil
	}

	if wfc.OnlyRunClientDefinedRevalidateFunc {
		c.must_reload_broadcast(
			refreshFilePayload{ChangeType: changeTypeRevalidate},
			must_reload_broadcast_opts{
				wait_for_app:  true,
				wait_for_vite: c.isUsingVite(),
				message:       "Running client-defined revalidate function",
			},
		)
		return nil
	}

	if !evtDetails.isWaveCSS || needsHardReloadEvenIfNonGo {
		c.must_reload_broadcast(
			refreshFilePayload{ChangeType: changeTypeOther},
			must_reload_broadcast_opts{
				wait_for_app:  true,
				wait_for_vite: c.isUsingVite(),
				message:       "Hard reloading browser",
			},
		)
		return nil
	}
	// At this point, we know it's a CSS file

	cssType := changeTypeNormalCSS
	if evtDetails.isCriticalCSS {
		cssType = changeTypeCriticalCSS
	}

	rfp := refreshFilePayload{
		ChangeType: cssType,

		// These must be called AFTER ProcessCSS
		CriticalCSS:  base64.StdEncoding.EncodeToString([]byte(c.GetCriticalCSS())),
		NormalCSSURL: c.GetStyleSheetURL(),
	}
	c.must_reload_broadcast(rfp, must_reload_broadcast_opts{
		wait_for_app:  false,
		wait_for_vite: false,
		message:       "Hot reloading browser (CSS)",
	})

	return nil
}

func (c *Config) callback(wfc *WatchedFile, evtDetails *EvtDetails) error {
	if evtDetails.isGo {
		return c.compile_go_binary()
	}

	if evtDetails.isWaveCSS {
		if getNeedsHardReloadEvenIfNonGo(wfc) {
			return c.runOtherFileBuild(wfc, evtDetails)
		}
		if evtDetails.isCriticalCSS {
			c.processCSSCritical()
		} else {
			c.processCSSNormal()
		}
	}

	return c.runOtherFileBuild(wfc, evtDetails)
}

// This is different than inside of handleGoFileChange, because here we
// assume we need to re-run other build steps too, not just recompile Go.
// Also, we don't necessarily recompile Go here (we only necessarily) run
// the other build steps. We only recompile Go if wfc.RecompileGoBinary is true.
func (c *Config) runOtherFileBuild(wfc *WatchedFile, evtDetails *EvtDetails) error {
	err := c.Build(BuildOptions{
		RecompileGoBinary:          wfc.RecompileGoBinary,
		IsDev:                      true,
		is_dev_rebuild:             true,
		just_run_simple_file_build: evtDetails.isWaveCSS || wfc.OnlyRunClientDefinedRevalidateFunc,
	})
	if err != nil {
		msg := fmt.Sprintf("error: failed to build app: %v", err)
		c.Logger.Error(msg)
		return errors.New(msg)
	}
	return nil
}
