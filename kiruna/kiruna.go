package kiruna

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/river-now/river/kiruna/internal/ki"
	"github.com/river-now/river/kit/middleware"
)

type (
	Kiruna      struct{ c *Config }
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

func New(c *ki.Config) *Kiruna {
	c.MainInit(ki.MainInitOptions{}, "kiruna.New")
	return &Kiruna{c}
}

// If you want to do a custom build command, just use
// Kiruna.BuildWithoutCompilingGo() instead of Kiruna.Build(),
// and then you can control your build yourself afterwards.

func (k Kiruna) Build() error {
	return k.c.Build(ki.BuildOptions{RecompileGoBinary: true})
}
func (k Kiruna) BuildWithoutCompilingGo() error {
	return k.c.Build(ki.BuildOptions{})
}

func (k Kiruna) GetPublicFS() (fs.FS, error) {
	return k.c.GetPublicFS()
}
func (k Kiruna) GetPrivateFS() (fs.FS, error) {
	return k.c.GetPrivateFS()
}
func (k Kiruna) MustGetPublicFS() fs.FS {
	fs, err := k.c.GetPublicFS()
	if err != nil {
		panic(err)
	}
	return fs
}
func (k Kiruna) MustGetPrivateFS() fs.FS {
	fs, err := k.c.GetPrivateFS()
	if err != nil {
		panic(err)
	}
	return fs
}
func (k Kiruna) GetPublicURL(originalPublicURL string) string {
	return k.c.GetPublicURL(originalPublicURL)
}
func (k Kiruna) MustGetPublicURLBuildtime(originalPublicURL string) string {
	return k.c.MustGetPublicURLBuildtime(originalPublicURL)
}
func (k Kiruna) MustStartDev() {
	k.c.MustStartDev()
}
func (k Kiruna) GetCriticalCSS() template.CSS {
	return template.CSS(k.c.GetCriticalCSS())
}
func (k Kiruna) GetStyleSheetURL() string {
	return k.c.GetStyleSheetURL()
}
func (k Kiruna) GetRefreshScript() template.HTML {
	return template.HTML(k.c.GetRefreshScript())
}
func (k Kiruna) GetRefreshScriptSha256Hash() string {
	return k.c.GetRefreshScriptSha256Hash()
}
func (k Kiruna) GetCriticalCSSElementID() string {
	return ki.CriticalCSSElementID
}
func (k Kiruna) GetStyleSheetElementID() string {
	return ki.StyleSheetElementID
}
func (k Kiruna) GetBaseFS() (fs.FS, error) {
	return k.c.GetBaseFS()
}
func (k Kiruna) GetCriticalCSSStyleElement() template.HTML {
	return k.c.GetCriticalCSSStyleElement()
}
func (k Kiruna) GetCriticalCSSStyleElementSha256Hash() string {
	return k.c.GetCriticalCSSStyleElementSha256Hash()
}
func (k Kiruna) GetStyleSheetLinkElement() template.HTML {
	return k.c.GetStyleSheetLinkElement()
}
func (k Kiruna) GetServeStaticHandler(addImmutableCacheHeaders bool) (http.Handler, error) {
	return k.c.GetServeStaticHandler(addImmutableCacheHeaders)
}
func (k Kiruna) MustGetServeStaticHandler(addImmutableCacheHeaders bool) http.Handler {
	handler, err := k.c.GetServeStaticHandler(addImmutableCacheHeaders)
	if err != nil {
		panic(err)
	}
	return handler
}
func (k Kiruna) GetPublicFileMap() (FileMap, error) {
	return k.c.GetPublicFileMap()
}
func (k Kiruna) GetPublicFileMapKeysBuildtime() ([]string, error) {
	return k.c.GetPublicFileMapKeysBuildtime()
}
func (k Kiruna) GetPublicFileMapElements() template.HTML {
	return k.c.GetPublicFileMapElements()
}
func (k Kiruna) GetPublicFileMapScriptSha256Hash() string {
	return k.c.GetPublicFileMapScriptSha256Hash()
}
func (k Kiruna) GetPublicFileMapURL() string {
	return k.c.GetPublicFileMapURL()
}
func (k Kiruna) SetupDistDir() {
	k.c.SetupDistDir()
}
func (k Kiruna) GetSimplePublicFileMapBuildtime() (map[string]string, error) {
	return k.c.GetSimplePublicFileMapBuildtime()
}
func (k Kiruna) GetPrivateStaticDir() string {
	return k.c.GetPrivateStaticDir()
}
func (k Kiruna) GetPublicStaticDir() string {
	return k.c.GetPublicStaticDir()
}
func (k Kiruna) GetPublicPathPrefix() string {
	return k.c.GetPublicPathPrefix()
}
func (k Kiruna) ViteProdBuild() error {
	return k.c.ViteProdBuild()
}
func (k Kiruna) GetViteManifestLocation() string {
	return k.c.GetViteManifestLocation()
}
func (k Kiruna) GetViteOutDir() string {
	return k.c.GetViteOutDir()
}
func (k Kiruna) BuildHelper(hook func(isDev bool) error) {
	k.c.BuildHelper(hook)
}
func (k Kiruna) GetRiverUIVariant() string {
	return k.c.GetRiverUIVariant()
}
func (k Kiruna) GetRiverHTMLTemplateLocation() string {
	return k.c.GetRiverHTMLTemplateLocation()
}
func (k Kiruna) GetRiverClientEntry() string {
	return k.c.GetRiverClientEntry()
}
func (k Kiruna) GetRiverClientRouteDefsFile() string {
	return k.c.GetRiverClientRouteDefsFile()
}
func (k Kiruna) GetRiverTSGenOutPath() string {
	return k.c.GetRiverTSGenOutPath()
}
func (k Kiruna) GetRiverPublicURLFuncName() string {
	return k.c.GetRiverPublicURLFuncName()
}
func (k Kiruna) GetRiverAutoETags() bool {
	return k.c.GetRiverAutoETags()
}
func (k Kiruna) GetConfigFile() string {
	return k.c.GetConfigFile()
}

// Forwards requests for "/favicon.ico" to "/{your-public-prefix}/favicon.ico".
// Not necessary if you're explicitly defining your favicon anywhere.
// Only comes into play if your preference is to drop a "favicon.ico" file into
// your public static directory and call it a day.
func (k Kiruna) FaviconRedirect() middleware.Middleware {
	return k.c.FaviconRedirect()
}

// __TODO the func re-definitions in this file are becoming unwieldy. Refactor.
