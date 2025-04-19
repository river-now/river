# 🏔️ Wave

[![Go Reference](https://pkg.go.dev/badge/github.com/river-now/river/wave.svg)](https://pkg.go.dev/github.com/river-now/river/wave)
[![Go Report Card](https://goreportcard.com/badge/github.com/river-now/river/wave)](https://goreportcard.com/report/github.com/river-now/river/wave)

Wave is a simple, powerful library for building and optimizing fullstack Go
applications, with live browser refresh and excellent dev-prod parity. It's sort
of like Vite, but for Go apps.

### Dev server features

- Automatic smart rebuilds and browser refreshes
- Instant hot reloading for CSS files (without a full page refresh)
- Highly configurable to support any use case
- Glob pattern file watching
- Granular build hooks with customizable timing strategies

### Production optimizations

- Static asset hashing and embedding
- Basic CSS bundling and minification
- Critical CSS inlining
- Safely serve public static assets with immutable cache headers

Dev-time reloads are smart and fast. Based on the type of file you edit and your
associated configuration options, Wave will do the minimum amount of work
necessary to get your changes to your browser as quickly as possible.

Wave has a few lightweight runtime helpers for referencing hashed static assets
from Go code and templates (e.g., `Wave.GetPublicURL("favicon.ico")`) and for
including your CSS in your HTML templates (e.g.,
`Wave.GetCriticalCSSStyleElement()`, `Wave.GetStyleSheetLinkElement()`). They
have zero third-party dependencies and are aggressively cached whenever
possible, so you can feel free to call them even in the hot path of your
application.

Wave is completely decoupled from any specific frameworks or libraries, so you
can use it with any Go server framework or router you choose, or just use the
standard library. Moreover, unlike some alternatives, Wave doesn't require you
to install any tooling on your machine. It is orchestrated solely from inside
your repo and its dependencies.

## Starter Tutorial From Scratch (~5 minutes)

Let's get a Wave project set up from scratch. This should only take a few
minutes to complete. The only prerequisite is that you have Go installed on your
machine.

### Scaffolding

Start by initializing a new Go module in an empty directory, replacing
`your-module-name` with your own module name:

```sh
go mod init your-module-name
```

Then run the following commands to create the necessary directories and files
for your project:

```sh
# Scaffold directories
mkdir -p cmd/app cmd/build cmd/dev
mkdir -p private-static public-static/prehashed
mkdir -p dist/wave internal/platform

# Create placeholder files
touch cmd/app/main.go cmd/build/main.go cmd/dev/main.go
touch private-static/index.go.html
touch critical.css main.css
touch dist/wave/x dist/dist.go internal/platform/wave.go
```

---

### Setup `dist/dist.go`

Now copy this into your `dist/dist.go` file:

```go
package dist

import "embed"

//go:embed wave
var FS embed.FS
```

---

### Setup `internal/platform/wave.go`

Now copy this into your `internal/platform/wave.go` file, replacing
`your-module-name` with your own module name:

```go
package platform

import (
	"your-module-name/dist"

	"github.com/river-now/river/wave"
)

var Wave = wave.New(&wave.Config{
	DistFS:           dist.FS,
	MainAppEntry:     "./cmd/app/main.go",
	DistDir:          "./dist",
	PrivateStaticDir: "./private-static",
	PublicStaticDir:  "./public-static",
	CriticalCSSEntry: "./critical.css",
	NormalCSSEntry:   "./main.css",
})
```

---

### Add Wave as a dependency

Now go get Wave and tidy up:

```sh
go get github.com/river-now/river
go mod tidy
```

---

### Setup `./private-static/index.go.html`

Now copy this into your `./private-static/index.go.html` file:

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    {{.Wave.GetCriticalCSSStyleElement}} {{.Wave.GetStyleSheetLinkElement}}
  </head>
  <body>
    <div>
      <h1>Hello, world!</h1>
      <p>Hello from "./private-static/index.go.html"</p>
    </div>
    {{.Wave.GetRefreshScript}}
  </body>
</html>
```

---

### Setup `cmd/app/main.go`

And now copy this into your `cmd/app/main.go` file, replacing `your-module-name`
with your own module name:

```go
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"your-module-name/internal/platform"

	"github.com/river-now/river/wave"
)

func main() {
	// Health check endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files from "dist/wave/static/public" directory, accessible at "/public/"
	http.Handle("/public/", platform.Wave.MustGetServeStaticHandler("/public/", true))

	// Serve an HTML file using html/template
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		privateFS, err := platform.Wave.GetPrivateFS()
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		tmpl, err := template.ParseFS(privateFS, "index.go.html")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct{ Wave *wave.Wave }{
			Wave: platform.Wave,
		})
		if err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
		}
	})

	port := wave.MustGetPort()

	fmt.Printf("Starting server on: http://localhost:%d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
```

---

### Setup `cmd/build/main.go`

And copy this into your `cmd/build/main.go` file, replacing `your-module-name`
with your own module name:

```go
package main

import "your-module-name/internal/platform"

func main() {
	err := platform.Wave.Build()
	if err != nil {
		panic(err)
	}
}
```

This file is what you'll want to run when you're ready to build for production.
Running `go run ./cmd/build` will build your project and save your binary to
`dist/main`. Assuming you used `DistFS` to embed your static assets, you can now
run your binary from anywhere on the build machine, and it will serve your
static assets from the embedded filesystem. If you chose not to embed your
static assets, you'll just need to make sure that the binary is a sibling of the
`dist/wave` directory in order to serve your static assets from disk. This will
work automatically if you just copy over your dist directory after a prod build.

> [!NOTE] Oftentimes you'll want to handle compilation of your Go binary
> yourself. In such cases, you can use `platform.Wave.BuildWithoutCompilingGo()`
> instead of `platform.Wave.Build()`. This will run all the same Wave-specific
> processing (static asset hashing, etc.) but will stop short of producing an
> executable.

---

### Setup `cmd/dev/main.go`

Now copy this into your `cmd/dev/main.go` file, replacing `your-module-name`
with your own module name:

```go
package main

import (
	"your-module-name/internal/platform"

	"github.com/river-now/river/wave"
)

func main() {
	platform.Wave.MustStartDev(&wave.DevConfig{
		HealthcheckEndpoint: "/healthz",
		WatchedFiles:        wave.WatchedFiles{{Pattern: "**/*.go.html"}},
	})
}
```

---

### Run the dev server

Now try running the dev server:

```sh
go run ./cmd/dev
```

If you copied everything correctly, you should see some logging, with a link to
your site on localhost, either at port `8080` or some fallback port. If you see
an error, double check that you copied everything correctly.

---

### Edit critical CSS

Now paste the following into your `./critical.css` file, and hit save:

```css
body {
  background-color: darkblue;
  color: white;
}
```

If you leave your browser open and your dev server running, you should see the
changes reflected in your browser nearly instantly via hot CSS reloading. Notice
that the CSS above is being inlined into your document head. This is because
your `CriticalCSSEntry` config option is set to `./critical.css`.

---

### Edit normal CSS

Now let's make sure your normal stylesheet is also working. Copy this into your
`./main.css` file:

```css
h1 {
  color: red;
}
```

When you hit save, this should also hot reload.

> [!NOTE] If you want to separate your CSS into multiple files, you can do so
> using CSS `@import` syntax. This also works for your critical CSS file.

---

### Edit your HTML template

Now let's try editing your html template at `./private-static/index.go.html`.

Find the line that says `<h1>Hello, world!</h1>` (line 10) and change it to:
`<h1 style="color: green;">Hello, world!</h1>`.

When you hit save, your browser page should automatically refresh itself. This
happens because of the `{Pattern: "**/*.go.html"}` item in the
`wave.WatchedFiles` slice in `cmd/dev/main.go`. If you were to remove that item
and restart your dev server, the page would not reload when you save your html
file (if you don't believe me, go give it a try).

When you want to watch different file types, you can add them to the
`wave.WatchedFiles` slice using glob patterns, and there are a whole bunch of
ways to tweak this to get your desired reload behavior and sequencing, including
callbacks and more. Feel free to explore your auto-complete options here or dive
into the Wave source code to learn more.

---

### Setup .gitignore

If desired, you can bootstrap a new `.gitignore` file by running the following:

```sh
echo "dist/*\n\!dist/dist.go" > .gitignore
```

## Alternatives

If you're just looking for automatic Go application rebuilds only, without
automatic browser refreshes or static asset build tooling, then Wave may be
overkill for you, and you could just use
<a href="https://github.com/cosmtrek/air" target="_blank">Air</a> instead.

That said, you can put Wave into a simpler `ServerOnly` mode if you want, which
skips the frontend-targeted build steps and just does automatic Go application
rebuilds.

In either case, one benefit of Wave over Air is that it doesn't require you to
install any tooling on your machine. It is orchestrated solely from inside your
repo and its dependencies. So when a new developer joins your team, they can
just clone your repo and be ready to rock as soon as they run `go mod tidy`
(instead of needing to install and configure Air on their machine first).

## Copyright and License

Copyright 2024 Samuel J. Cook. Licensed under the BSD 3-Clause License.
