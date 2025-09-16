---
title: River Documentation
---

## Creating a new River app

`npm create river@latest`

- Works in monorepos
- Can create new directory or use current directory
- Can create new Go module or use existing parent module
- Supports `React`, `Preact`, and `Solid`
- Supports `npm`, `pnpm`, `yarn`, and `bun`
- Supports `Tailwind` styling or traditional CSS
- Supports `Vercel` deployment target or standard self-hosted

---

## Client APIs

### Package Exports

#### river.now/client

##### Listening for Build ID Changes

River will automatically hard reload the page when it discovers there has been a
new build if it is safe to do so (navigations and revalidations). Sometimes,
however, River can't know whether it's safe to hard reload the page (namely,
upon any API query response or failed API mutation response). To handle those
cases, you can listen for new build ID events yourself using
`addBuildIDListener`.

```ts
import { addBuildIDListener } from "river.now/client";

addBuildIDListener(({ oldID, newID }) => {
	// do something, such as:
	// - hard reload
	// - show a toast to the user
	// - save data to localStorage, then hard reload
	// - etc.
});
```

**NOTE:** If you're deploying to Vercel,
[Skew Protection](blog/vercel-skew-protection) can make this even better.

##### Listening for Location Events

You probably won't need to do this unless you're doing something goofy, but if
you do ever need to listen for location changes, you can use
`addLocationListener`.

```ts
import { addLocationListener, getLocation } from "river.now/client";

addLocationListener(() => {
	const location = getLocation();
	// do whatever you need
});
```

You probably don't need this because River provides reactive primitives (built
on top of `addLocationListener` and `getLocation`) that are more convenient and
higher-level: a `location` signal is exported from `river.now/preact` and
`river.now/solid`, and a `useLocation` hook is exported from `river.now/react`:

```ts
import { location } from "river.now/preact"; // Preact signal
import { location } from "river.now/solid"; // Solid signal
import { useLocation } from "river.now/react"; // Jotai-powered hook
```

**NOTE:** `addLocationListener` triggers if you update search params directly
with the underlying `npm:history` instance (available via `getHistoryInstance`
exported from `river.now/client`), as well as on River-controlled navigations
(given that River uses `npm:history` under the hood). It does NOT, however,
trigger when you call `window.history.pushState` directly (so don't do that
unless you really know what you're doing). This differs from
`addRouteChangeListener`, which only triggers when you do a River-controlled
navigation.

- [x] addBuildIDListener
- [x] addLocationListener
- [ ] addRouteChangeListener
- [ ] addStatusListener
- [ ] getBuildID
- [ ] getHistoryInstance
- [ ] getLocation
- [ ] getRootEl
- [ ] getStatus
- [ ] initClient
- [ ] revalidate
- [ ] riverNavigate
- [ ] submit
- [ ] revalidateOnWindowFocus
- [ ] setupGlobalLoadingIndicator
- [ ] buildMutationURL
- [ ] buildQueryURL
- [ ] getRouterData
- [ ] makeTypedNavigate
- [ ] RouteProps
- [ ] props.Outlet

#### river.now/{ui-lib}

- makeTypedAddClientLoader
- makeTypedUseLoaderData
- makeTypedUsePatternLoaderData
- makeTypedUseRouterData
- makeTypedLink
- RiverLink
- location (preact/solid) / useLocation (react)
- RiverRootOutlet
- RiverProvider (react)

#### river.now/vite

- riverVitePlugin

### Typed API Client

### Typed App Utils

### How matching works

### Explicit vs. Implicit Index Segments

### Styling

- Critical Styles
- Non-Critical Styles
- Tailwind

### river.gen.ts

### routes.ts & registration of frontend routes

---

## Server APIs

- IsJSONRequest
- NewHeadEls
- RiverBuildIDHeaderKey
- EnableThirdPartyRouter
- AdHocType
- BuildOptions
- HeadEl
- River
    - Fields
        - Wave
        - GetDefaultHeadEls
        - GetHeadElUniqueRules
        - GetRootTemplateData
    - Methods
        - Build
        - Init
        - GetActionsHandler
        - GetLoadersHandler
        - GetCurrentBuildID
        - IsCurrentBuildJSONRequest

### App-Level Utils

- ActionsRouter
- ActionCtx
- NewAction
- LoadersRouter
- LoaderCtx
- NewLoader

### Core Router

- control.Wave.ServeStatic
- healthcheck
- GetLoadersHandler
- GetActionsHandler
- EnableThirdPartyRouter

---

## Assets

### HTML Entry Template

- RiverHeadEls
- RiverSSRScript
- RiverRootID
- RiverBodyScripts

### Public vs. private assets

- How to bypass hashing
- How to reference assets from backend
- How to reference assets from frontend
    - Build-time (recommended -- hashed filename directly in build)
    - Run-time (discouraged -- increases bundle size since we have to ship a
      map) -- waveRuntimeURL (exported from river.gen.ts)
- Bonus: referencing assets on frontend is type safe

---

## Control Layer

### Build

- ExtraTSCode
- AdHocTypes

### Serve

### Dist

### river.config.go

- River instance
- Wave instance
    - Embedding config + embedding static FS (must include embed directive)

### wave.config.json

---

## Crucial Kit Packages

- kit/headels
- kit/matcher
- kit/mux
- kit/response (proxy)
- kit/tasks
- kit/validate

---

## If deploying to Vercel

---

## If using Tailwind

---

## If a different core router (like Chi)

```

```
