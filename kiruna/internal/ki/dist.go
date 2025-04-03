package ki

import (
	"github.com/sjc5/river/kit/dirs"
)

const (
	PUBLIC  = "public"
	PRIVATE = "private"
)

type Dist struct {
	Binary *dirs.File
	Static *dirs.Dir[DistStatic]
}

type DistStatic struct {
	Assets   *dirs.Dir[DistStaticAssets]
	Internal *dirs.Dir[DistKirunaInternal]
	Keep     *dirs.File
}

type DistStaticAssets struct {
	Public  *dirs.Dir[DistStaticAssetsPublic]
	Private *dirs.DirEmpty
}

type DistStaticAssetsPublic struct {
	PublicInternal *dirs.DirEmpty
}

type DistKirunaInternal struct {
	CriticalDotCSS             *dirs.File
	NormalCSSFileRefDotTXT     *dirs.File
	PublicFileMapFileRefDotTXT *dirs.File
}

func toDistLayout(cleanDistDir string) *dirs.Dir[Dist] {
	x := dirs.Build(cleanDistDir, dirs.ToRoot(Dist{
		Binary: dirs.ToFile("main"),
		Static: dirs.ToDir("static", DistStatic{
			Assets: dirs.ToDir("assets", DistStaticAssets{
				Public: dirs.ToDir(PUBLIC, DistStaticAssetsPublic{
					PublicInternal: dirs.ToDirEmpty("internal"),
				}),
				Private: dirs.ToDirEmpty(PRIVATE),
			}),
			Internal: dirs.ToDir("internal", DistKirunaInternal{
				CriticalDotCSS:             dirs.ToFile("critical.css"),
				NormalCSSFileRefDotTXT:     dirs.ToFile("normal_css_file_ref.txt"),
				PublicFileMapFileRefDotTXT: dirs.ToFile("public_file_map_file_ref.txt"),
			}),
			Keep: dirs.ToFile(".keep"),
		}),
	}))

	return x
}
