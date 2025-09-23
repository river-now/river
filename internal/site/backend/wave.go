package backend

import (
	"embed"

	"github.com/river-now/river/kit/fsutil"
	"github.com/river-now/river/wave"
)

//go:embed all:dist/static wave.config.json
var embedFS embed.FS

var Wave = wave.New(wave.Config{
	WaveConfigJSON: fsutil.MustReadFile(embedFS, "wave.config.json"),
	DistStaticFS:   fsutil.MustSub(embedFS, "dist", "static"),
})
