package ki

import (
	"path/filepath"

	"github.com/river-now/river/kit/viteutil"
)

func (c *Config) isUsingVite() bool {
	return c._uc.Vite != nil
}

func (c *Config) GetViteManifestLocation() string {
	return filepath.Join(c._uc.Core.StaticAssetDirs.Private, "river_out", "river_vite_manifest.json")
}

func (c *Config) GetViteOutDir() string {
	return filepath.Join(c._uc.Core.StaticAssetDirs.Public, "prehashed")
}

func (c *Config) toViteCtx() *viteutil.BuildCtx {
	return viteutil.NewBuildCtx(&viteutil.BuildCtxOptions{
		JSPackageManagerBaseCmd: c._uc.Vite.JSPackageManagerBaseCmd,
		JSPackageManagerCmdDir:  c._uc.Vite.JSPackageManagerCmdDir,
		OutDir:                  c.GetViteOutDir(),
		ManifestOut:             c.GetViteManifestLocation(),
		ViteConfigFile:          c._uc.Vite.ViteConfigFile,
	})
}

func (c *Config) viteDevBuild() (*viteutil.BuildCtx, error) {
	if !c.isUsingVite() {
		return nil, nil
	}
	ctx := c.toViteCtx()
	err := ctx.DevBuild()
	return ctx, err
}

func (c *Config) ViteProdBuild() error {
	if !c.isUsingVite() {
		return nil
	}
	ctx := c.toViteCtx()
	return ctx.ProdBuild()
}
