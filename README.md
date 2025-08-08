# River

<img src="internal/site/frontend/__static/river-banner.webp" alt="River logo banner">

## Introduction

River is a Go / TypeScript meta-framework with first-class support for React,
Solid, and Preact – built on Vite.

River provides the benefits of modern web development frameworks without the
downsides of isomorphic JavaScript and component hydration patterns.

Core Features:

- Seamless Vite integration
- Nested routing with concurrent Go-based loaders
- Blazing fast TypeScript type generation
- Hybrid-SSR architecture
- React, Preact, and Solid support

Optional Extras:

- Global critical CSS inlining
- Link pre-fetching on hover
- Ability to deeply integrate the build and dev refresh cycle with any upstream
  dependencies in any language, via our Wave build tool
- Much, much more

While River currently supports React, Preact, and Solid, 95% of River's client
architecture just uses vanilla browser APIs, making it pretty trivial to support
any client UI library while still providing the same underlying development
model. Even our Link components are light wrappers over a vanilla core.

Our core philosophy is simple: You don't need backend JS to build a fully modern
and performant web application.

Yes, it is possible to get great performance and SEO scores, a fully type-safe
application, and a frictionless full-stack developer experience, all without
backend JS.

Now, River doesn't think that all backend JS is bad _per se_. TypeScript in
particular is a wonderfully dynamic language, and it can be great for writing
many backend services. But what we _do_ reject is the modern myth that you need
to use a full SSR and hydration or component streaming model to achieve the
requirements of a modern full-stack web application. It just doesn't need to be
that complex.

Now, let me explain the hybrid-SSR architecture and how it recreates the
benefits of traditional full SSR without technically doing full SSR.

It's quite simple really. The hybrid-SSR architecture has four ingredients:

1. Fully populated document head with all necessary and appropriate tags and
   data, generated dynamically on a per-route basis.
2. Pre-loaded route data immediately available for your client components to
   consume.
3. Properly hashed, immutably edge-cached, and parallel-loaded static
   dependencies.
4. Client-rendered document body inside the dynamic outer shell (ready to
   execute and paint almost immediately thanks to items 1-3 above).

The fourth item (client-rendered document body) is what will make many of you
skeptical about this architecture. That's OK! We welcome skepticism.

We feel that once you try it out and deploy an app or two using the River
hybrid-SSR architecture, you'll realize that as long as you have items 1-3 in
place, item 4 is not actually a real-world problem. We think you'll be reminded
that client-side rendering is really nice, actually (especially when it sits on
top of the dynamic scaffolding you get with River). It greatly simplifies your
application architecture, frees you up to use cheaper and more performant
backend languages, and rids you of those annoying hydration errors.
Win-win-win-win-win. You get the _actual_ benefits of full SSR without the
downsides.

OK, so we have covered initial page loads. What about subsequent navigations?
Well, we just do the same thing, except with a smaller JSON payload.

We know all your dependencies (including both JS and CSS) at a route level, so
it's trivial to preload it all and block navigations for a few milliseconds
until everything is loaded and ready. And if you choose to use link pre-fetching
on hover, the route transition will feel instant for anyone with a decent
internet connection.

Additionally, all navigations are performed using a single request. No
waterfalls or jank. No request-per-route-segment model. Just a fast navigation
that feels solid and familiar to users.

To sum up, River provides many benefits over popular full-SSR frameworks.
Faster, cheaper server. Smaller all-around payloads. No hydration errors. No
subtle JS runtime incompatabilities. Yet it also provides many of the same
benefits, just using a different architecture. Excellent SEO. Wonderful and
blazing fast user experience. Fully type-safe and frictionless developer
experience. All yours for the taking.

## Installation

## Go

```sh
go get github.com/river-now/river
```

## TypeScript

```sh
npm i -D river.now
```

Why a dev dependency? Because it is more explicit about what is actually
happening. The `"dependencies"`/`"devDependencies"` distinction in your
`package.json` is a NodeJS concept, not a browser JavaScript concept. For
explicitness, and because River's npm modules are targeted at browsers (not
NodeJS), River projects will usually declare all dependencies as dev
dependencies to make it clear that they are being transformed before being
shipped to the browser.
