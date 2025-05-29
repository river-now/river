package markdown

import (
	"app"
	"io"

	"github.com/adrg/frontmatter"
	"github.com/river-now/river/kit/xyz/fsmarkdown"
	"github.com/river-now/river/wave"
	"github.com/russross/blackfriday/v2"
)

var Markdown = fsmarkdown.New(fsmarkdown.Options{
	FS:                app.Wave.MustGetPrivateFS(),
	FrontmatterParser: func(r io.Reader, v any) ([]byte, error) { return frontmatter.Parse(r, v) },
	MarkdownParser: func(b []byte) []byte {
		return blackfriday.Run(b, blackfriday.WithExtensions(blackfriday.AutoHeadingIDs|blackfriday.CommonExtensions))
	},
	IsDev: wave.GetIsDev(),
})
