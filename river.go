package river

import (
	"github.com/river-now/river/internal/framework"
	"github.com/river-now/river/kit/htmlutil"
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
