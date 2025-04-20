---
title: Get Started
description: Get started with the river.now framework
---

## Caveats

Before we get started, please understand that River is in an alpha state. You
should use it in production **_if and only if_**:

1. You know what you're doing;
2. You pin your versions; and
3. You're willing to fix breaking (and likely unspecified) changes when you
   update versions.

Buyer beware.

OK, let's get started...

## Intro

This guide is going to walk you through manually bootstrapping a new River
project from scratch. Why are we doing it this way? Well, mainly because we
don't have a CLI-based bootstrapper yet. But also because it's nice to see how
all the pieces fit together (let's pretend that's the real reason).

We are going to use React for this guide, but you can totally use Solid or
Preact if you want. You will need to change a few things here and there, but I'm
confident you can figure that out. If you get stuck, feel free to ask for help
via a GitHub issue.

## Guide

### pre-reqs

Make sure you have both `Go (>= 1.24)` and `Node (>= 22.11)` installed on your
machine.

### wave.config.json

River is configured through a `wave.config.json` file. Wave is a first-party,
lower-level build tool used by River. It doesn't matter where this file lives,
but usually you'll want it to be in your project root.

Let's create our Wave config file:

```sh
touch wave.config.json
```

Then add this to it:

```json
{
  "$schema": "__dist/static/internal/schema.json",
  "Core": {
    "ConfigLocation": "wave.config.json",
    "DevBuildHook": "go run ./__cmd/build --dev --hook",
    "ProdBuildHook": "go run ./__cmd/build --hook",
    "MainAppEntry": "__cmd/app",
    "DistDir": "__dist",
    "StaticAssetDirs": {
      "Private": "backend/__static",
      "Public": "frontend/__static"
    },
    "CSSEntryFiles": {
      "Critical": "frontend/css/main.critical.css",
      "NonCritical": "frontend/css/main.css"
    },
    "PublicPathPrefix": "/public/"
  },
  "River": {
    "UIVariant": "react",
    "HTMLTemplateLocation": "entry.go.html",
    "ClientEntry": "frontend/entry.tsx",
    "ClientRouteDefsFile": "frontend/routes.ts",
    "TSGenOutPath": "frontend/river.gen.ts",
    "BuildtimePublicURLFuncName": "waveURL",
    "AutoETags": true
  },
  "Vite": {
    "JSPackageManagerBaseCmd": "npx"
  },
  "Watch": {
    "HealthcheckEndpoint": "/healthz"
  }
}
```

### vite.config.ts

River is deeply integrated with `vite`. Let's create a Vite config file:

```sh
touch vite.config.ts
```

Then add this to it:

```ts
import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "vite";
import { riverVitePlugin } from "./frontend/river.gen.js";

export default defineConfig({
  plugins: [react(), riverVitePlugin()],
});
```

### tsconfig.json

River is a TypeScript-first framework. Let's create a `tsconfig.json` file:

```sh
touch tsconfig.json
```

Then add this to it:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "forceConsistentCasingInFileNames": true,
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true,
    "esModuleInterop": true,
    "noUncheckedIndexedAccess": true,
    "verbatimModuleSyntax": true,
    "allowImportingTsExtensions": true,
    "jsx": "react-jsx"
  },
  "exclude": ["node_modules"]
}
```

### package.json

At this point, you get the pattern. From here on out, I'm going to give you
touch commands along with content to add to the newly created file, and I'm
going to stop explaining that every time.

```sh
touch package.json
```

```json
{
  "type": "module",
  "scripts": {
    "dev": "go run ./__cmd/build --dev",
    "build": "go run ./__cmd/build"
  },
  "devDependencies": {
    "@types/react": "^19.1.2",
    "@types/react-dom": "^19.1.2",
    "@vitejs/plugin-react-swc": "^3.9.0",
    "jotai": "^2.12.3",
    "react": "^19.1.0",
    "react-dom": "^19.1.0",
    "river.now": "0.17.15",
    "typescript": "^5.8.3",
    "vite": "^6.3.2"
  }
}
```

### Init Go module

Let's initialize our Go project:

```sh
go mod init app
go get github.com/river-now/river
```

### app.go

```sh
touch app.go
```

```go
package app

import (
	"embed"
	"net/http"

	"github.com/river-now/river"
	"github.com/river-now/river/wave"
)

//go:embed wave.config.json
var configBytes []byte

//go:embed all:__dist/static
var staticFS embed.FS

var Wave = wave.New(&wave.Config{
	ConfigBytes:            configBytes,
	StaticFS:               staticFS,
	StaticFSEmbedDirective: "all:__dist/static",
})

var River = &river.River{
	Wave:                 Wave,
	GetDefaultHeadEls: getDefaultHeadEls,
	GetRootTemplateData:  getRouteTemplateData,
}

func getDefaultHeadEls(r *http.Request) ([]*river.HeadEl, error) {
	blocks := []*river.HeadEl{{
		Tag:       "title",
		InnerHTML: "River",
	}}
	return blocks, nil
}

func getRouteTemplateData(r *http.Request) (map[string]any, error) {
	data := map[string]any{
		"Lang": "en",
	}
	return data, nil
}
```

**NOTE:** You don't have to use `go:embed` if you don't want. Using `go:embed`
just makes it so you only need to deploy your final binary. If you don't use
`go:embed`, just make sure that the `static` folder generated in your `DistDir`
is a sibling to your final binary when you serve your app.

### .gitignore

```sh
touch .gitignore
```

```sh
# Node
**/node_modules/

# Wave
__dist/static/*
!__dist/static/.keep
__dist/main*
```

### Directories

Now let's create some directories we will need.

```sh
mkdir __cmd/app -p
mkdir __cmd/build
mkdir __dist/static -p
mkdir backend/__static -p
mkdir backend/router
mkdir backend/server
mkdir frontend/__static -p
mkdir frontend/components/routes -p
mkdir frontend/css
```

### frontend/entry.tsx

```sh
touch frontend/entry.tsx
```

```tsx
import { createRoot } from "react-dom/client";
import { getRootEl, initClient } from "river.now/client";
import { RiverRootOutlet } from "river.now/react";

await initClient(() => {
  const el = getRootEl();
  const root = createRoot(el);
  root.render(<RiverRootOutlet />);
});
```

### frontend/routes.ts

This is a special file. It is used only during the build process as a way to
statically define where your root modules live and what their export names are.
If you don't define an export name, it's assumed to be a default export.

**IMPORTANT:** Don't import anything into this file, and don't try to
dynamically generate module or export name strings. It won't work!

```sh
touch frontend/routes.ts
```

```ts
import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;

routes.Register("/", {
  module: import("./components/routes/home.tsx"),
  export: "Home",
});

export default routes;
```

### frontend/css/main.critical.css

```sh
touch frontend/css/main.critical.css
```

```css
html {
  background: black;
}
```

**NOTE:** Anything you put in your critical CSS file will be inlined into the
document head when your app serves its initial HTML payload. This is great for
preventing annoying flash-of-unstyled-content (`FOUC`) and content-layout-shift
(`CLS`) issues.

### frontend/css/main.css

```sh
touch frontend/css/main.css
```

```css
html {
  color: white;
  font-family: monospace;
}
```

## // TBC

```go
// to be continued
```
