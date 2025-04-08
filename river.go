package river

import (
	"github.com/sjc5/river/internal/framework"
	"github.com/sjc5/river/kit/htmlutil"
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
