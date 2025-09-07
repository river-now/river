---
title: Docs
description: Documentation for the River framework
---

## Bootstrapping a New River Project

To start a new River project, run the following command in your terminal:

`npm create river@latest`

After ensuring you have the required minimum versions of `Go` and `Node`
installed, the bootstrapper CLI will ask you a series of questions about how you
want to configure your River project.

## Choosing a Frontend UI Library

During the bootstrapping process, you will be asked to choose from `React`,
`Preact`, and `Solid` for your frontend UI library.

With one small exception (explained below), River's APIs are **_identical_**
across all three options.

### Solid-Specific API Exception

When reading the current route's location data in a `React` or `Preact` project,
River provides a `useLocation` hook. In a `Solid` project, however, such a hook
would be redundant. Instead, `river.now/solid` is able to directly provide a
`location` accessor, skipping the `useLocation()` indirection entirely. Here's
what I mean:

In React (or Preact), we need to call `useLocation()`:

```tsx
import { useLocation } from "river.now/react";

function Component() {
	const location = useLocation();
	return <div>Current path: {location.pathname}</div>;
}
```

In Solid, we can just use the `location` accessor directly:

```tsx
import { location } from "river.now/solid";

function Component() {
	return <div>Current path: {location().pathname}</div>;
}
```

## Generated Types

The foundation of River's backend-frontend type safety is its generated types
file.

By convention, the bootstrapper will create this file at
`./frontend/river.gen.ts` (aka, your "**_River types file_**"), but you can
configure it to live wherever you want (and be named whatever you want) by
editing the `River.TSGenOutPath` field in your `./app/wave.config.json` file
(aka, your "**_Wave config file_**").

Your River types file contains several things:

- All your loader and action route definitions, with input/output types,
  patterns, and param keys
- All your ad hoc types that are used by your routes and/or otherwise generated
  by your backend (see [Sharing Ad Hoc Types](#sharing-ad-hoc-types) below).
- CONTINUE

## Sharing Ad Hoc Types

If you want to share additional ad hoc types to your frontend from your Go
backend (that aren't already being used by any River routes), you can do so by
passing an instance of them to the `river.BuildOptions.AdHocTypes` slice in your
`./cmd/build/main.go` file (aka, your "**_River build script_**").

## River Build Script

By convention, your River build script lives at `./cmd/build/main.go`, but you
can put it wherever you want by editing your Wave config file.
