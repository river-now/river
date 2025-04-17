package ki

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
)

type fileVal struct {
	Val         string
	IsPrehashed bool
}

type FileMap map[string]fileVal

func (c *Config) GetServeStaticHandler(addImmutableCacheHeaders bool) (http.Handler, error) {
	publicFS, err := c.GetPublicFS()
	if err != nil {
		errMsg := fmt.Sprintf("error getting public FS: %v", err)
		c.Logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}
	if addImmutableCacheHeaders {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			http.StripPrefix(c.GetPublicPathPrefix(), http.FileServer(http.FS(publicFS))).ServeHTTP(w, r)
		}), nil
	}
	return http.StripPrefix(c.GetPublicPathPrefix(), http.FileServer(http.FS(publicFS))), nil
}

func (c *Config) getInitialPublicFileMapFromGobBuildtime() (FileMap, error) {
	return c.loadMapFromGob(PublicFileMapGobName, true)
}

func (c *Config) getInitialPublicFileMapFromGobRuntime() (FileMap, error) {
	return c.loadMapFromGob(PublicFileMapGobName, false)
}

func (c *Config) MustGetPublicURLBuildtime(originalPublicURL string) string {
	fileMapFromGob, err := c.getInitialPublicFileMapFromGobBuildtime()
	if err != nil {
		c.Logger.Error(fmt.Sprintf(
			"error getting public file map from gob (buildtime) for originalPublicURL %s: %v", originalPublicURL, err,
		))
		panic(err)
	}

	url, err := c.getInitialPublicURLInner(originalPublicURL, fileMapFromGob)
	if err != nil {
		c.Logger.Error(fmt.Sprintf(
			"error getting initial public URL (buildtime) for originalPublicURL %s: %v", originalPublicURL, err,
		))
		panic(err)
	}

	return url
}

func (c *Config) getInitialPublicURL(originalPublicURL string) (string, error) {
	fileMapFromGob, err := c.runtime_cache.public_filemap_from_gob.Get()
	if err != nil {
		c.Logger.Error(fmt.Sprintf(
			"error getting public file map from gob for originalPublicURL %s: %v", originalPublicURL, err,
		))
		return path.Join(c._uc.Core.PublicPathPrefix, originalPublicURL), err
	}

	return c.getInitialPublicURLInner(originalPublicURL, fileMapFromGob)
}

func (c *Config) getInitialPublicURLInner(originalPublicURL string, fileMapFromGob FileMap) (string, error) {
	if strings.HasPrefix(originalPublicURL, "data:") {
		return originalPublicURL, nil
	}

	if hashedURL, existsInFileMap := fileMapFromGob[cleanURL(originalPublicURL)]; existsInFileMap {
		return path.Join(c._uc.Core.PublicPathPrefix, hashedURL.Val), nil
	}

	// If no hashed URL found, return the original URL
	c.Logger.Info(fmt.Sprintf(
		"GetPublicURL: no hashed URL found for %s, returning original URL",
		originalPublicURL,
	))

	return path.Join(c._uc.Core.PublicPathPrefix, originalPublicURL), nil
}

func publicURLsKeyMaker(x string) string { return x }

func (c *Config) GetPublicURL(originalPublicURL string) string {
	url, _ := c.runtime_cache.public_urls.Get(originalPublicURL)
	return url
}

func cleanURL(url string) string {
	return strings.TrimPrefix(path.Clean(url), "/")
}
