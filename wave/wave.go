package wave

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/river-now/river/kit/middleware"
	"github.com/river-now/river/wave/internal/ki"
)

type (
	Wave        struct{ c *Config }
	Config      = ki.Config
	FileMap     = ki.FileMap
	WatchedFile = ki.WatchedFile
	OnChangeCmd = ki.OnChangeHook
)

const (
	OnChangeStrategyPre              = ki.OnChangeStrategyPre
	OnChangeStrategyConcurrent       = ki.OnChangeStrategyConcurrent
	OnChangeStrategyConcurrentNoWait = ki.OnChangeStrategyConcurrentNoWait
	OnChangeStrategyPost             = ki.OnChangeStrategyPost
	PrehashedDirname                 = ki.PrehashedDirname
)

var (
	MustGetPort  = ki.MustGetAppPort
	GetIsDev     = ki.GetIsDev
	SetModeToDev = ki.SetModeToDev
)

func New(c *ki.Config) *Wave {
	c.MainInit(ki.MainInitOptions{}, "wave.New")
	return &Wave{c}
}

// If you want to do a custom build command, just use
// Wave.BuildWithoutCompilingGo() instead of Wave.Build(),
// and then you can control your build yourself afterwards.

func (k Wave) Build() error {
	return k.c.Build(ki.BuildOptions{RecompileGoBinary: true})
}
func (k Wave) BuildWithoutCompilingGo() error {
	return k.c.Build(ki.BuildOptions{})
}

func (k Wave) GetPublicFS() (fs.FS, error) {
	return k.c.GetPublicFS()
}
func (k Wave) GetPrivateFS() (fs.FS, error) {
	return k.c.GetPrivateFS()
}
func (k Wave) MustGetPublicFS() fs.FS {
	fs, err := k.c.GetPublicFS()
	if err != nil {
		panic(err)
	}
	return fs
}
func (k Wave) MustGetPrivateFS() fs.FS {
	fs, err := k.c.GetPrivateFS()
	if err != nil {
		panic(err)
	}
	return fs
}
func (k Wave) GetPublicURL(originalPublicURL string) string {
	return k.c.GetPublicURL(originalPublicURL)
}
func (k Wave) MustGetPublicURLBuildtime(originalPublicURL string) string {
	return k.c.MustGetPublicURLBuildtime(originalPublicURL)
}
func (k Wave) MustStartDev() {
	k.c.MustStartDev()
}
func (k Wave) GetCriticalCSS() template.CSS {
	return template.CSS(k.c.GetCriticalCSS())
}
func (k Wave) GetStyleSheetURL() string {
	return k.c.GetStyleSheetURL()
}
func (k Wave) GetRefreshScript() template.HTML {
	return template.HTML(k.c.GetRefreshScript())
}
func (k Wave) GetRefreshScriptSha256Hash() string {
	return k.c.GetRefreshScriptSha256Hash()
}
func (k Wave) GetCriticalCSSElementID() string {
	return ki.CriticalCSSElementID
}
func (k Wave) GetStyleSheetElementID() string {
	return ki.StyleSheetElementID
}
func (k Wave) GetBaseFS() (fs.FS, error) {
	return k.c.GetBaseFS()
}
func (k Wave) GetCriticalCSSStyleElement() template.HTML {
	return k.c.GetCriticalCSSStyleElement()
}
func (k Wave) GetCriticalCSSStyleElementSha256Hash() string {
	return k.c.GetCriticalCSSStyleElementSha256Hash()
}
func (k Wave) GetStyleSheetLinkElement() template.HTML {
	return k.c.GetStyleSheetLinkElement()
}
func (k Wave) GetServeStaticHandler(addImmutableCacheHeaders bool) (http.Handler, error) {
	return k.c.GetServeStaticHandler(addImmutableCacheHeaders)
}
func (k Wave) MustGetServeStaticHandler(addImmutableCacheHeaders bool) http.Handler {
	handler, err := k.c.GetServeStaticHandler(addImmutableCacheHeaders)
	if err != nil {
		panic(err)
	}
	return handler
}

func (k Wave) ServeStatic(addImmutableCacheHeaders bool) func(http.Handler) http.Handler {
	return k.c.ServeStaticPublicAssets(addImmutableCacheHeaders)
}
func (k Wave) GetPublicFileMap() (FileMap, error) {
	return k.c.GetPublicFileMap()
}
func (k Wave) GetPublicFileMapKeysBuildtime() ([]string, error) {
	return k.c.GetPublicFileMapKeysBuildtime()
}
func (k Wave) GetPublicFileMapElements() template.HTML {
	return k.c.GetPublicFileMapElements()
}
func (k Wave) GetPublicFileMapScriptSha256Hash() string {
	return k.c.GetPublicFileMapScriptSha256Hash()
}
func (k Wave) GetPublicFileMapURL() string {
	return k.c.GetPublicFileMapURL()
}
func (k Wave) SetupDistDir() {
	k.c.SetupDistDir()
}
func (k Wave) GetSimplePublicFileMapBuildtime() (map[string]string, error) {
	return k.c.GetSimplePublicFileMapBuildtime()
}
func (k Wave) GetPrivateStaticDir() string {
	return k.c.GetPrivateStaticDir()
}
func (k Wave) GetPublicStaticDir() string {
	return k.c.GetPublicStaticDir()
}
func (k Wave) GetPublicPathPrefix() string {
	return k.c.GetPublicPathPrefix()
}
func (k Wave) ViteProdBuild() error {
	return k.c.ViteProdBuild()
}
func (k Wave) GetViteManifestLocation() string {
	return k.c.GetViteManifestLocation()
}
func (k Wave) GetViteOutDir() string {
	return k.c.GetViteOutDir()
}
func (k Wave) Builder(hook func(isDev bool) error) {
	k.c.Builder(hook)
}
func (k Wave) GetRiverUIVariant() string {
	return k.c.GetRiverUIVariant()
}
func (k Wave) GetRiverHTMLTemplateLocation() string {
	return k.c.GetRiverHTMLTemplateLocation()
}
func (k Wave) GetRiverClientEntry() string {
	return k.c.GetRiverClientEntry()
}
func (k Wave) GetRiverClientRouteDefsFile() string {
	return k.c.GetRiverClientRouteDefsFile()
}
func (k Wave) GetRiverTSGenOutPath() string {
	return k.c.GetRiverTSGenOutPath()
}
func (k Wave) GetRiverBuildtimePublicURLFuncName() string {
	return k.c.GetRiverBuildtimePublicURLFuncName()
}
func (k Wave) GetConfigFile() string {
	return k.c.GetConfigFile()
}
func (k Wave) GetDistDir() string {
	return k.c.GetDistDir()
}
func (k Wave) GetStaticPrivateOutDir() string {
	return k.c.GetStaticPrivateOutDir()
}
func (k Wave) GetStaticPublicOutDir() string {
	return k.c.GetStaticPublicOutDir()
}

// Forwards requests for "/favicon.ico" to "/{your-public-prefix}/favicon.ico".
// Not necessary if you're explicitly defining your favicon anywhere.
// Only comes into play if your preference is to drop a "favicon.ico" file into
// your public static directory and call it a day.
func (k Wave) FaviconRedirect() middleware.Middleware {
	return k.c.FaviconRedirect()
}
