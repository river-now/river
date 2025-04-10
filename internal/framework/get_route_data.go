package framework

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sjc5/river/kit/errutil"
	"github.com/sjc5/river/kit/headblocks"
	"github.com/sjc5/river/kit/htmlutil"
	"github.com/sjc5/river/kit/mux"
	"golang.org/x/sync/errgroup"
)

var (
	errNotFound   = errors.New("not found")
	isErrNotFound = errutil.ToIsErrFunc(errNotFound)
)

type UIRouteOutput struct {
	HasRootData bool    `json:"hasRootData,omitempty"`
	LoadersData []any   `json:"loadersData,omitempty"`
	LoadersErrs []error `json:"loadersErrs,omitempty"`

	Params      mux.Params  `json:"params,omitempty"`
	SplatValues SplatValues `json:"splatValues,omitempty"`

	Title string              `json:"title,omitempty"`
	Meta  []*htmlutil.Element `json:"metaHeadBlocks,omitempty"`
	Rest  []*htmlutil.Element `json:"restHeadBlocks,omitempty"`

	// LoadersErrorMessages []string            `json:"loadersErrorMessages,omitempty"`
	OutermostErrorIndex int `json:"outermostErrorIndex,omitempty"`

	ImportURLs []string `json:"importURLs,omitempty"`
	ExportKeys []string `json:"exportKeys,omitempty"`
	Deps       []string `json:"deps,omitempty"`
	CSSBundles []string `json:"cssBundles,omitempty"`

	ViteDevURL string `json:"viteDevURL,omitempty"`
}

type getUIRouteDataOutput struct {
	uiRouteOutput *UIRouteOutput
	didRedirect   bool
}

func (h *River) getUIRouteData(w http.ResponseWriter, r *http.Request,
	nestedRouter *mux.NestedRouter,
) (*getUIRouteDataOutput, error) {

	tasksCtx := nestedRouter.TasksRegistry().NewCtxFromRequest(r)

	eg := errgroup.Group{}

	var defaultHeadBlocks []*htmlutil.Element
	var err error

	eg.Go(func() error {
		if h.GetDefaultHeadBlocks != nil {
			defaultHeadBlocks, err = h.GetDefaultHeadBlocks(r)
			if err != nil {
				errMsg := fmt.Sprintf("could not get default head blocks: %v", err)
				Log.Error(errMsg)
				return errors.New(errMsg)
			}
		} else {
			defaultHeadBlocks = []*htmlutil.Element{}
		}
		return nil
	})

	uiRoutesData := h.getUIRoutesData(w, r, nestedRouter, tasksCtx)
	if !uiRoutesData.found {
		return nil, errNotFound
	}

	if uiRoutesData.didRedirect {
		return &getUIRouteDataOutput{didRedirect: true}, nil
	}

	activePathData := uiRoutesData.activePathData

	err = eg.Wait()
	if err != nil {
		Log.Error(err.Error())
		return nil, err
	}

	var hb []*htmlutil.Element
	hb = make([]*htmlutil.Element, 0, len(activePathData.HeadBlocks)+len(defaultHeadBlocks))
	hb = append(hb, defaultHeadBlocks...)
	hb = append(hb, activePathData.HeadBlocks...)

	// dedupe and organize into HeadBlocks struct
	headBlocks := headblocks.ToHeadBlocks(hb)

	uiRouteOutput := &UIRouteOutput{
		HasRootData: activePathData.HasRootData,
		LoadersData: activePathData.LoadersData,
		LoadersErrs: activePathData.LoadersErrs,

		Params:      activePathData.Params,
		SplatValues: activePathData.SplatValues,

		Title: headBlocks.Title,
		Meta:  headBlocks.Meta,
		Rest:  headBlocks.Rest,

		// LoadersErrorMessages: activePathData.LoadersErrMsgs,
		OutermostErrorIndex: activePathData.OutermostErrorIndex,

		ImportURLs: activePathData.ImportURLs,
		ExportKeys: activePathData.ExportKeys,
		Deps:       activePathData.Deps,
		CSSBundles: h.getCSSBundles(activePathData.Deps),

		ViteDevURL: h.getViteDevURL(),
	}

	return &getUIRouteDataOutput{uiRouteOutput: uiRouteOutput}, nil
}
