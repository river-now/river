package ki

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/river-now/river/kit/executil"
)

func (c *Config) get_is_using_embedded_fs() bool {
	return c.StaticFS != nil
}

func (c *Config) get_initial_base_dir_fs() (fs.FS, error) {
	return os.DirFS(c._dist.S().Static.FullPath()), nil
}

func (c *Config) getSubFSPrivate() (fs.FS, error) { return c.__getSubFS(PRIVATE) }
func (c *Config) getSubFSPublic() (fs.FS, error)  { return c.__getSubFS(PUBLIC) }

// subDir = "public" or "private"
func (c *Config) __getSubFS(subDir string) (fs.FS, error) {
	// __LOCATION_ASSUMPTION: Inside "dist/static"
	_path := path.Join(c._dist.S().Static.S().Assets.LastSegment(), subDir)

	baseFS, err := c.GetBaseFS()
	if err != nil {
		wrapped := fmt.Errorf("error getting %s FS: %w", subDir, err)
		c.Logger.Error(wrapped.Error())
		return nil, wrapped
	}
	subFS, err := fs.Sub(baseFS, _path)
	if err != nil {
		wrapped := fmt.Errorf("error getting %s FS: %w", subDir, err)
		c.Logger.Error(wrapped.Error())
		return nil, wrapped
	}
	return subFS, nil
}

func (c *Config) GetPublicFS() (fs.FS, error) {
	return c.runtime_cache.public_fs.Get()
}

func (c *Config) GetPrivateFS() (fs.FS, error) {
	return c.runtime_cache.private_fs.Get()
}

// GetBaseFS returns a filesystem interface that works across different environments (dev/prod)
// and supports both embedded and non-embedded filesystems.
func (c *Config) GetBaseFS() (fs.FS, error) {
	return c.runtime_cache.base_fs.Get()
}

func (c *Config) get_initial_base_fs() (fs.FS, error) {
	// DEV
	// There is an expectation that you run the dev server from the root of your project,
	// where your go.mod file is.
	if GetIsDev() {
		return os.DirFS(c._dist.S().Static.FullPath()), nil
	}

	// If we are using the embedded file system, we should use the dist file system
	if c.get_is_using_embedded_fs() {
		directive := c.StaticFSEmbedDirective

		if directive == "" {
			// sanity check -- should never happen (downstream of user config validation)
			panic("StaticFSEmbedDirective is empty, cannot use embedded FS")
		}

		// if first 4 are "all:", strip
		if len(directive) > 4 && directive[:4] == "all:" {
			directive = directive[4:]
		}

		// Navigate into the embedded directory structure specified by StaticFSEmbedDirective.
		embeddedFS, err := fs.Sub(c.StaticFS, directive)
		if err != nil {
			return nil, err
		}

		return embeddedFS, nil
	}

	// If we are not using the embedded file system, we should use the os file system,
	// and assume that the executable is a sibling to the wave-outputted "static" directory
	execDir, err := executil.GetExecutableDir()
	if err != nil {
		return nil, err
	}

	return os.DirFS(path.Join(execDir, c._dist.S().Static.LastSegment())), nil
}
