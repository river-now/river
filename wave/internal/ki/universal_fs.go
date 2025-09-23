package ki

import (
	"fmt"
	"io/fs"
	"os"
	"path"
)

func (c *Config) get_is_using_embedded_fs() bool {
	return c.DistStaticFS != nil
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

	fsToUse := c.DistStaticFS
	if fsToUse == nil {
		panic("DistStaticFS is nil in production mode; you must provide an embedded FS")
	}

	return fsToUse, nil
}
