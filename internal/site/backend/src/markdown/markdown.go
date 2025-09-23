package markdown

import (
	"io"
	"site/backend"

	"github.com/adrg/frontmatter"
	"github.com/river-now/river/kit/lab/fsmarkdown"
	"github.com/russross/blackfriday/v2"
)

var Markdown = fsmarkdown.New(fsmarkdown.Options{
	FS:                backend.App.MustGetPrivateFS(),
	FrontmatterParser: func(r io.Reader, v any) ([]byte, error) { return frontmatter.Parse(r, v) },
	MarkdownParser: func(b []byte) []byte {
		return blackfriday.Run(b, blackfriday.WithExtensions(blackfriday.AutoHeadingIDs|blackfriday.CommonExtensions))
	},
	IsDev: backend.App.GetIsDev(),
})
