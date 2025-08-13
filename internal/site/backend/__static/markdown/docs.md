---
title: Docs
description: Frequently asked questions about the river.now framework
---

**NOTE:** These docs are currently just a bunch of sparse random topics. A comprehensive overhaul is coming soon.

## Creating a new project

To create a new project, open a terminal and run the following commands (adjusted as appropriate for your preferred package manager):

```bash
npm create river@latest
npm run dev
```

## Nested Routing

### What changes when you use explicit index segments?

#### Explicit Index Segments

When you **_do_** use explicit index segments, `"/"` will **_always_** be
matched, and `"_index"` (assuming that's your explicit index segment) will only
be matched when you are at the literal home slash route. Similarly, the pattern
`"/foo"` becomes in essence a layout route. If you also want a default index
route at that path, you would add the pattern `"/foo/_index"`.

#### Implicit Index Segments (default)

When you **_do not_** use explicit index segments, `"/"` will **_only_** be
matched when you are at the literal home slash route, while `""` (empty string)
will **_always_** be matched. The pattern `"/foo"` is a layout route (just like
when you use explicit index segments). However, instead of doing something like
`"/foo/_index"` for an index route, you'd just add a trailing slash, like this:
`"/foo/"`.

#### Which is better?

In most cases, it's better and less confusing to use explicit index segments
because a single trailing slash can be easy to miss.

### How do loader errors work?

#### Returning an error message to the user

If you want to return an error message to the user (which will render the error
component associated with a route segment), you should return an error from your
loader. This will return a 200 OK, and any "parent" loaders that did not error
will still render their applicable loader data (and any children will
necessarily not render, given that their parent errored).

#### Making the navigation request fail completely

If you straight up want the request to fail, you can do so via the
`c.ResponseProxy().SetStatus(code, statusMsg)` helper available in your loader
context. In this case, it doesn't matter what your loader returns. The server
will just return the error status and the navigation simply won't work. In most
cases, this will not be what you want, but sometimes (access-related invariants,
perhaps) this will be useful.

## Framework-controlled Route Data

To access framework-controlled route data, you should make your own local
`useRouterData` hook, like so:

```ts
import { makeTypedUseRouterData } from "river.now/react";
import type { RiverLoader, RiverRootData } from "./river.gen.ts";

export const useRouterData = makeTypedUseRouterData<
	RiverLoader,
	RiverRootData
>();
```

Then you can call that in any component:

```ts
const routerData = useRouterData();
```

If you want the route params to be strongly typed, you can either pass your
route props in (assuming _they_ are strongly typed) or pass a pattern type arg:

```ts
// strongly typed from the props
const routerData = useRouterData(props);

// strongly typed from the pattern type arg
const routerData2 = useRouterData<"/:myDynamicSegment">();
```

This returns an object containing your current `buildID`, an array of currently
matched route patterns (`matchedPatterns`), an array of the current route's
splat values, if any (`splatValues`), an object mapping the current route's
dynamic param keys (if any) to their values (`params`), and `rootData`, which is
whatever loader data is returned from your always-matched route segment, if
applicable ("" or "/" if using an explicit index segment).

## Uncontrolled Location Data

### Subscribe to Location Changes

If you need to subscribe to location data (_i.e._, the current URL's `pathname`,
`search`, and `hash`), you can use the `useLocation` hook exported from
`river.now/${ui-lib}`, like so:

```ts
import { useLocation } from "river.now/react";

// in a component:
const location = useLocation();
```

For Solid, it's already a reactive `Accessor`, so having a hook is unnecessary.
So it's this instead:

```ts
import { location } from "river.now/solid";

// then call the accessor anywhere in a component:
location();
```

### Low-Level History Stack

If you need deeper access to the history stack (_e.g._, if you want to update
search params to store client state without doing a server navigation), you can
get direct access to the underlying `npm:history` instance using
`getHistoryInstance()` exported from `river.now/client`. Then, you can do
whatever you want with it, such as:

```ts
import { getHistoryInstance } from "river.now/client";

const history = getHistoryInstance();

history.replace({ search: "?test=123" });
```

Then, if you need to react to such changes, you can use the `useLocation` hook
mentioned above.
