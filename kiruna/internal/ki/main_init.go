package ki

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sjc5/river/kit/colorlog"
	"github.com/sjc5/river/kit/safecache"
	"golang.org/x/sync/semaphore"
)

/////////////////////////////////////////////////////////////////////
/////// MAIN INIT
/////////////////////////////////////////////////////////////////////

type MainInitOptions struct {
	IsDev     bool
	IsRebuild bool
}

func (c *Config) MainInit(opts MainInitOptions, calledFrom string) {
	var err error
	var file []byte

	// LOGGER
	if c.Logger == nil {
		c.Logger = colorlog.New("kiruna")
	}

	c.fileSemaphore = semaphore.NewWeighted(100)

	// USER CONFIG
	if file, err = os.ReadFile(c.ConfigFile); err != nil {
		c.panic("failed to read user config", err)
	}
	c._uc = new(UserConfig)
	if err = json.Unmarshal(file, c._uc); err != nil {
		c.panic("failed to unmarshal user config", err)
	}

	// CLEAN SOURCES
	c.cleanSources = CleanSources{
		Dist:          filepath.Clean(c._uc.Core.DistDir),
		PrivateStatic: filepath.Clean(c._uc.Core.StaticAssetDirs.Private),
		PublicStatic:  filepath.Clean(c._uc.Core.StaticAssetDirs.Public),
	}
	if c._uc.Core.CSSEntryFiles.Critical != "" {
		c.cleanSources.CriticalCSSEntry = filepath.Clean(c._uc.Core.CSSEntryFiles.Critical)
	}
	if c._uc.Core.CSSEntryFiles.NonCritical != "" {
		c.cleanSources.NonCriticalCSSEntry = filepath.Clean(c._uc.Core.CSSEntryFiles.NonCritical)
	}

	// DIST LAYOUT
	c._dist = toDistLayout(c.cleanSources.Dist)

	c.InitRuntimeCache()

	// AFTER HERE, ALL DEV-TIME STUFF
	if !opts.IsDev {
		return
	}

	c.dev.mu.Lock()
	defer c.dev.mu.Unlock()

	c.kill_browser_refresh_mux()

	c._rebuild_cleanup_chan = make(chan struct{})

	c.cleanWatchRoot = filepath.Clean(c._uc.Watch.WatchRoot)

	SetModeToDev()

	// HEALTH CHECK ENDPOINT
	if c._uc.Watch.HealthcheckEndpoint == "" {
		c._uc.Watch.HealthcheckEndpoint = "/"
	}

	if !opts.IsRebuild {
		c.browserTabManager = newClientManager()
		go c.browserTabManager.start()
	}

	c.ignoredFilePatterns = []string{
		c.get_binary_output_path(),
	}

	c.naiveIgnoreDirPatterns = []string{
		"**/.git",
		"**/node_modules",
		c._dist.S().Static.FullPath(),
		filepath.Join(c.cleanSources.PublicStatic, noHashPublicDirsByVersion[0]),
		filepath.Join(c.cleanSources.PublicStatic, noHashPublicDirsByVersion[1]),
	}

	for _, p := range c.naiveIgnoreDirPatterns {
		c.ignoredDirPatterns = append(c.ignoredDirPatterns, filepath.Join(c.cleanWatchRoot, p))
	}
	for _, p := range c._uc.Watch.Exclude.Dirs {
		c.ignoredDirPatterns = append(c.ignoredDirPatterns, filepath.Join(c.cleanWatchRoot, p))
	}
	for _, p := range c._uc.Watch.Exclude.Files {
		c.ignoredFilePatterns = append(c.ignoredFilePatterns, filepath.Join(c.cleanWatchRoot, p))
	}

	c.defaultWatchedFiles = []WatchedFile{
		{Pattern: filepath.Join(c.cleanSources.PrivateStatic, "**/*"), RestartApp: true},
		{Pattern: filepath.Join(c.cleanSources.PublicStatic, "**/*"), RestartApp: true},
	}

	// Loop through all WatchedFiles...
	for i, wfc := range c._uc.Watch.Include {
		// and make each WatchedFile's Pattern relative to cleanWatchRoot...
		c._uc.Watch.Include[i].Pattern = filepath.Join(c.cleanWatchRoot, wfc.Pattern)
		// then loop through such WatchedFile's OnChangeHooks...
		for j, oc := range wfc.OnChangeHooks {
			// and make each such OnChangeCallback's ExcludedPatterns also relative to cleanWatchRoot
			for k, p := range oc.Exclude {
				c._uc.Watch.Include[i].OnChangeHooks[j].Exclude[k] = filepath.Join(c.cleanWatchRoot, p)
			}
		}
	}

	c.matchResults = safecache.NewMap(c.get_initial_match_results, c.match_results_key_maker, nil)

	if c.watcher != nil {
		if err := c.watcher.Close(); err != nil {
			c.panic("failed to close watcher", err)
		}
		c.watcher = nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.panic("failed to create watcher", err)
	}

	c.watcher = watcher

	if err := c.add_directory_to_watcher(c.cleanWatchRoot); err != nil {
		c.panic("failed to add directory to watcher", err)
	}
}
