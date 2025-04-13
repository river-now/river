package river

import (
	_ "embed"

	"github.com/river-now/river/internal/framework"
	"github.com/river-now/river/kit/htmlutil"
	"github.com/river-now/river/kit/parseutil"
)

/////////////////////////////////////////////////////////////////////
/////// PUBLIC API
/////////////////////////////////////////////////////////////////////

type (
	River        = framework.River
	HeadBlock    = htmlutil.Element
	AdHocType    = framework.AdHocType
	BuildOptions = framework.BuildOptions
)

var (
	GetIsJSONRequest = framework.GetIsJSONRequest
)

//go:embed package.json
var packageJSON string

// This utility exists primarily in service of the River.now
// website. There is no guarantee that this utility will always
// be available or kept up to date.
func Internal__GetCurrentNPMVersion() string {
	_, _, currentVersion := parseutil.PackageJSONFromString(packageJSON)
	return currentVersion
}
