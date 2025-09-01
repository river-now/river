# River Documentation

## What is River?

River is a full-stack Go web framework that provides type-safe communication
between a Go backend and TypeScript frontend, with first-class support for
React, Preact, and Solid. It features Remix-style nested routing and is deeply
integrated with [Vite](https://vite.dev).

## Wave Build Tool

The River framework uses a lower-level build tool called Wave. Wave is extremely
flexible and supports almost any dev-time file watching setup you could desire
(_e.g._, building a Rust crate into a WASM file and placing it into your static
assets folder to serve to your frontend). It has comprehensive support for
watching and ignoring whatever files and directories you want, and then taking
actions at precise times in the build cycle when those files are modified. To
learn how to configure the Wave build tool, please see
[the Wave docs](/docs/wave).

## Getting Started

### Creating a River App

```bash
npm create river@latest
```

The above command will run the River bootstrapper CLI, which will guide you
through a series of questions and then generate a complete full-stack River
application with sensible defaults that you can modify, including:

- Configured routers for loaders and actions
- TypeScript client setup
- Build configuration
- Example routes and components
- Automatic input validation setup
- `nprogress` loading bar, hooked into River data lifecycles

Then run:

```bash
cd your-app # If you created a new directory from the CLI
npm run dev # Adjust for your preferred package manager
```

## Core Concepts

### Routing Patterns

River uses pattern matching for routes:

```go
// Pattern examples (assuming an explicit index segment of `_index`)
"/"                 // Root route
"/about"            // Static route
"/users/:id"        // Dynamic parameter (`:` is configurable)
"/files/*"          // Splat – catches everything after (`*` is configurable)
"/_index"           // Explicit index (when configured)
```

**Pattern Matching Examples:**

- `/files/*` matches `/files/docs/readme.md`, with the following `splatValues`:
  `["docs", "readme.md"]`
- `/users/:id/posts/:postId` matches `/users/123/posts/456`, with the following
  `params`: `{id: "123", postId: "456"}`

### Index Segments

River supports two approaches for index routes:

#### Explicit Index Segments (RECOMMENDED)

```go
LoadersRouter := mux.NewNestedRouter(&mux.NestedOptions{
    ExplicitIndexSegment: "_index",
})
```

With this configuration:

- `"/"` matches ALL routes (acts as a layout route for your whole app)
- `"/_index"` only matches the home route `/`
- `/about` is a layout, while `/about/_index` is the index page

#### Implicit Index Segments:

```go
LoadersRouter := mux.NewNestedRouter(&mux.NestedOptions{
    // ExplicitIndexSegment not set (defaults to "")
})
```

With this configuration:

- `""` (empty string) matches ALL routes (acts as a layout route for your whole
  app)
- `"/"` only matches the home route
- `/about` is a layout, while `/about/` (with a trailing slash) is the index
  page

Because trailing slashes can be easy to miss, we recommend just using explicit
index segments.

## Server-Side Development

### Loaders (Data Fetching)

River's `mux.NestedRouter` handles nested route matching and executes registered
task handlers in parallel, merging their response proxies according to specific
rules.

The bootstrapper generates a convenient wrapper for you:

```go
// backend/router/loaders.go
var LoadersRouter = mux.NewNestedRouter(&mux.NestedOptions{
    ExplicitIndexSegment: "_index",
})

type LoaderCtx struct {
    *mux.NestedReqData
    // You can add custom fields here
}

func NewLoader[O any](
	pattern string, f func(c *LoaderCtx) (O, error),
) *mux.TaskHandler[mux.None, O] {
    // ...
}
```

Using the bootstrapper's setup:

```go
type RootData struct {
    Message  string
    Count    int
}

var _ = NewLoader("/", func(c *LoaderCtx) (*RootData, error) {
    return &RootData{
        Message: "Hello",
        Count:   42,
    }, nil
})
```

### Actions (API Endpoints)

Actions handle API requests and mutations. Using the below setup, actions will
be available at `/api/*` (automatically handled by frontend API client).

```go
var ActionsRouter = mux.NewRouter(&mux.Options{
    MountRoot: "/api/",
    ParseInput: func(r *http.Request, inputPtr any) error {
        // The bootstrapper sets this up with River's validate helpers
        if r.Method == http.MethodGet {
            return validate.URLSearchParamsInto(r, inputPtr)
        }
        return validate.JSONBodyInto(r, inputPtr)
    },
})

// Helper pattern (optional)
type ActionCtx[I any] struct {
    *mux.ReqData[I]
	// You can add custom fields here
}

func NewAction[I any, O any](
	method, pattern string, f func(c *ActionCtx[I]) (O, error),
) *mux.TaskHandler[I, O] {
    // ...
}
```

Simple action without input:

```go
var _ = NewAction("GET", "/status", func(
	c *ActionCtx[mux.None],
) (string, error) {
    return "ok", nil
})
```

Action with input (feed the desired input type into the ActionCtx generic):

```go
type CreateUserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var _ = NewAction("POST", "/users", func(
	c *ActionCtx[CreateUserInput],
) (*User, error) {
    // c.Input() contains parsed and validated input
    return createUser(c.Input()), nil
})
```

### Input Parsing and Validation

**_What River always does:_** River calls the `ParseInput` function you provide
(if any) before your handler executes.

**_What the bootstrapper sets up for you:_** The generated code includes a
`ParseInput` that uses River's `validate` package helpers.

```go
ParseInput: func(r *http.Request, inputPtr any) error {
    if r.Method == http.MethodGet {
        return validate.URLSearchParamsInto(r, inputPtr)
    }
    return validate.JSONBodyInto(r, inputPtr)
}
```

How the validate helpers work:

1. `validate.JSONBodyInto` / `validate.URLSearchParamsInto` parse the request
2. They check if the input type has a `Validate()` method
3. If it does, they call it (also recursively calls `Validate()` on any nested
   types)

How `ParseInput()` works:

1. If `ParseInput` returns an error, parsing fails, and River's `mux` then sends
   400 Bad Request to the client (along with the error message returned from
   `ParseInput` -- so make sure that's client safe)
2. If `ParseInput` succeeds, River feeds the result into your handler to be
   consumed via `c.Input()`

Example with validation:

```go
type UserInput struct {
    Name string `json:"name"`
}

func (i *UserInput) Validate() error {
    if i.Name == "" {
        return errors.New("name required")
    }
    return nil
}

var _ = NewAction("POST", "/users", func(c *ActionCtx[UserInput]) (*User, error) {
    // This never executes if validation failed
    // River already sent 400 to the client with "name required"
    return createUser(c.Input()), nil
})
```

### Error Handling

River handles errors very differently for loaders versus actions.

#### Loader Errors

Loaders can handle errors in two ways:

**1. Return an error → Error boundary displays**

```go
var _ = NewLoader("/protected", func(c *LoaderCtx) (*Data, error) {
    if !isAuthenticated(c.Request()) {
        return nil, errors.New("authentication required")
    }
    return fetchData(), nil
})
```

What happens:

- Response status: 200 OK
- Parent loaders that succeeded: Their data is sent
- Child loaders: Are discarded
- Client receives: JSON with `outermostError` field containing your error
  message
- UI behavior: Error boundary component renders with the message returned from
  the loader `error`

**IMPORTANT**: Whatever you return as an error from your loader WILL be sent to
the client. Do not return sensitive info there. The final error that you
propogate/return from your loader should be client-safe.

**2. Set error status via `ResponseProxy` → Navigation fails**

```go
var _ = NewLoader("/admin", func(c *LoaderCtx) (*Data, error) {
    if !isAdmin(c.Request()) {
        c.ResponseProxy().SetStatus(403, "Forbidden")
        return nil, nil // Don't return an error
    }
    return fetchData(), nil
})
```

What happens:

- Response status: `403` (or whatever you set)
- Client receives: HTTP error with no JSON data
- UI behavior: Navigation fails, current page stays

**You cannot combine these approaches.** If you set an error status (400+) via
ResponseProxy, the client gets an HTTP error and no error boundary will display,
even if you also return an error.

#### Action Errors

Actions have three distinct error behaviors:

**1. Validation errors → 400 with your message**

When using the validate helpers and validation fails:

```go
type CreatePostInput struct {
    Title string `json:"title"`
}

func (i *CreatePostInput) Validate() error {
    if i.Title == "" {
        return errors.New("title is required")  // Client sees this
    }
    return nil
}

var _ = NewAction("POST", "/posts", func(c *ActionCtx[CreatePostInput]) (*Post, error) {
    // Never reached if validation fails
    return createPost(c.Input()), nil
})
```

Client gets: 400 Bad Request with body "title is required"

**2. Regular errors → 500 without your message**

When you return a non-validation error from an action:

```go
var _ = NewAction("POST", "/update", func(c *ActionCtx[Input]) (*Output, error) {
    if somethingBadHappened {
        return nil, errors.New("database connection failed")
    }
    return result, nil
})
```

Client gets: 500 Internal Server Error with body "Internal Server Error" Your
actual error message is logged server-side but NOT sent to the client.

**3. Custom status via ResponseProxy**

For full control over the response:

```go
var _ = NewAction("POST", "/update", func(c *ActionCtx[Input]) (*Output, error) {
    if resourceExists {
        c.ResponseProxy().SetStatus(409, "Resource already exists")
        return nil, nil  // Don't return an error!
    }
    return result, nil
})
```

Client gets: 409 Conflict with body "Resource already exists"

**Critical:** In actions, don't return an error after setting a status with
ResponseProxy. If you do, the error will override your custom status with a 500:

```go
// WRONG - becomes 500 instead of 409
c.ResponseProxy().SetStatus(409)
return nil, errors.New("conflict")

// CORRECT - preserves 409
c.ResponseProxy().SetStatus(409, "Conflict")
return nil, nil
```

### Response Proxy

The ResponseProxy allows modifying the HTTP response:

```go
func (c *LoaderCtx) example() {
    proxy := c.ResponseProxy()

    // Headers
    proxy.SetHeader("Cache-Control", "max-age=3600")  // Replaces
    proxy.AddHeader("X-Custom", "value")              // Appends

    // Cookies
    proxy.SetCookie(&http.Cookie{Name: "theme", Value: "dark"})

    // Status
    proxy.SetStatus(403, "Forbidden")

    // Redirects
    proxy.Redirect(c.Request(), "/login")
}
```

**Merging Rules (when multiple loaders/middleware use ResponseProxy):**

1. **Status**: First error (4xx/5xx) wins; otherwise last success (2xx)
2. **Redirects**: First redirect wins (unless error occurred first)
3. **Headers**: Applied in order; SetHeader replaces, AddHeader appends
4. **Cookies**: Later cookies with same name overwrite earlier ones

### Middleware

River supports both HTTP middleware (traditional) and Task middleware
(parallel):

```go
// HTTP Middleware - runs sequentially
mux.SetGlobalHTTPMiddleware(router, corsMiddleware)
mux.SetMethodLevelHTTPMiddleware(router, "POST", authMiddleware)

// Task Middleware - runs in parallel
authMw := mux.TaskMiddlewareFromFunc(func(rd *mux.ReqData[mux.None]) (*User, error) {
    // Each task middleware gets its own ResponseProxy
    token := rd.Request().Header.Get("Authorization")
    user, err := validateToken(token)
    if err != nil {
        rd.ResponseProxy().SetStatus(401, "Unauthorized")
        return nil, err
    }
    return user, nil
})
mux.SetGlobalTaskMiddleware(router, authMw)

// Conditional middleware
mux.SetGlobalHTTPMiddleware(router, rateLimit, &mux.MiddlewareOptions{
    If: func(r *http.Request) bool {
        return strings.HasPrefix(r.URL.Path, "/api/")
    },
})
```

### Head Elements

Manage HTML `<head>` elements with deduplication:

```go
// Define uniqueness rules
var River = &river.River{
    GetHeadElUniqueRules: func() *headels.HeadEls {
        e := river.NewHeadEls(2)
        e.Meta(e.Property("og:title"))  // Only one og:title allowed
        return e
    },
    GetDefaultHeadEls: func(r *http.Request) ([]*htmlutil.Element, error) {
        // Default elements for all pages
        e := river.NewHeadEls()
        e.Title("My App")
        return e.Collect(), nil
    },
}

// Add page-specific elements in loaders
var _ = NewLoader("/article/:id", func(c *LoaderCtx) (*Article, error) {
    article := fetchArticle(c.Params()["id"])

    c.ResponseProxy().AddHeadElement(&htmlutil.Element{
        Tag: "meta",
        AttributesKnownSafe: map[string]string{
            "property": "og:title",
            "content":  article.Title,  // Overrides default
        },
    })

    return article, nil
})
```

## Client-Side Development

### Defining Routes

In `frontend/routes.ts`:

```typescript
import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;
export default routes;

// routes.Add(pattern, importPromise, componentKey, errorBoundaryKey?)
routes.Add("/", import("./root.tsx"), "Root");
routes.Add("/_index", import("./home.tsx"), "Home", "HomeError");
routes.Add("/users/:id", import("./user.tsx"), "User", "UserError");
```

### Route Components

```typescript
// user.tsx
import type { RouteProps } from "./types";

export function User(props: RouteProps<"/users/:id">) {
  const data = useLoaderData(props); // Typed from your Go loader
  const router = useRouterData(props);

  return (
    <div>
      <h1>User {router.params.id}</h1>
      <p>{data.name}</p>
      <props.Outlet /> {/* Render child routes */}
    </div>
  );
}

// Error boundary (optional)
export function UserError(props: { error: string }) {
  return <div>Error: {props.error}</div>;
}
```

### Error Boundaries

Error boundaries catch loader errors and display fallback UI. They only work
when the loader returns an error with a 200 OK response, not for HTTP errors.

```typescript
// Define in routes.ts (4th parameter)
routes.Add(
  "/dashboard",
  import("./dashboard.tsx"),
  "Dashboard",
  "DashboardError"
);

// dashboard.tsx
export function DashboardError(props: { error: string }) {
  const isAuthError = props.error.includes("authentication");

  if (isAuthError) {
    return (
      <div>
        <p>Please log in</p>
        <AppLink pattern="/login">Go to Login</AppLink>
      </div>
    );
  }

  return <div>Something went wrong: {props.error}</div>;
}
```

**Error Boundary Behavior:**

- When a loader returns an error, that route segment shows its error boundary
- Parent routes that loaded successfully continue to display
- Child routes never execute
- The app shell and parent layouts remain visible

### Data Hooks

```typescript
// useLoaderData - get current route's loader data
export function About(props: RouteProps<"/about">) {
  const data = useLoaderData(props); // Typed from your loader
  return <h1>{data.title}</h1>;
}

// useRouterData - get route context
export function Dynamic(props: RouteProps<"/users/:id">) {
  const router = useRouterData(props);
  // router.params.id - typed based on pattern
  // router.splatValues - for splat routes
  // router.rootData - from root loader
  // router.buildID - current build
  // router.matchedPatterns - active patterns
}

// usePatternLoaderData - get data from specific pattern
export function Child(props: RouteProps<"/parent/child">) {
  const childData = useLoaderData(props);
  const parentData = usePatternLoaderData("/parent"); // May be undefined
}
```

### Client Loaders

Client loaders run in the browser, in parallel with server loaders once
discovered:

```typescript
// Define and name your hook
const useClientData = addClientLoader("/_index", async (props) => {
  // props.params - route parameters
  // props.splatValues - splat values
  // props.serverDataPromise - server loader data
  // props.signal - AbortSignal

  const res = await fetch("/api/data", { signal: props.signal });
  return res.json();
});

// Use with props - data guaranteed
export function Home(props: RouteProps<"/_index">) {
  const clientData = useClientData(props); // Never undefined
  return <div>{clientData}</div>;
}

// Use without props - may be undefined
function OtherComponent() {
  const clientData = useClientData(); // May be undefined
  if (!clientData) return null;
  return <div>{clientData}</div>;
}
```

### Navigation

#### Links

```typescript
<AppLink pattern="/users/:id" params={{ id: "123" }}>
  View User
</AppLink>

<AppLink
  pattern="/files/*"
  splatValues={["docs", "readme.md"]}
  prefetch="intent"     // Prefetch on hover/focus
  scrollToTop={true}    // Scroll after navigation
  replace={true}        // Replace history entry
>
  View File
</AppLink>
```

#### Programmatic Navigation

```typescript
import { appNavigate } from "./app_utils";

await appNavigate({
	pattern: "/users/:id",
	params: { id: "123" },
	scrollToTop: true,
});
```

#### Manual Revalidation

You can manually trigger revalidation of the current route's data:

```typescript
import { revalidate } from "river.now/client";

await revalidate();
```

#### Navigation and Status Tracking

River provides several APIs to track navigation state and location changes:

```typescript
import {
  addStatusListener,
  getStatus,
  addLocationListener,
  getLocation,
  addBuildIDListener
} from "river.now/client";

// Track navigation/submission/revalidation status
const status = getStatus(); // { isNavigating, isSubmitting, isRevalidating }

addStatusListener((event) => {
  const { isNavigating, isSubmitting, isRevalidating } = event.detail;
  console.log("Status changed:", event.detail);
});

// Track location changes
addLocationListener(() => {
  const location = getLocation(); // { pathname, search, hash }
  console.log("Location changed:", location);
});

// Track build ID changes (useful for update notifications)
addBuildIDListener((event) => {
  const { oldID, newID, fromGETAction } = event.detail;
  if (oldID !== newID) {
    console.log("New version available!");
  }
});

// For React/Preact -- reactive location hook
import { useLocation } from "river.now/react"; // or river.now/preact
function Component() {
  const location = useLocation(); // Reactively tracks location
  return <div>Current path: {location.pathname}</div>;
}
```

### API Client

The API client generated by the bootstrapper provides type-safe access to
actions:

#### Queries (GET requests)

```typescript
import { api } from "./api_client";

const result = await api.query({
	pattern: "/get-user",
	input: { id: "123" }, // type-safe
});

if (result.success) {
	console.log(result.data); // type-safe
} else {
	console.error(result.error);
}
```

#### Mutations (POST, PUT, DELETE, etc.)

For POST mutations:

```typescript
const result = await api.mutate({
	pattern: "/create-user",
	input: { name: "Alice", email: "alice@example.com" },
	options: {
		revalidate: false, // Skip auto-revalidation
		dedupeKey: "create-user", // Prevent duplicate requests
	},
});
```

For non-POST mutations, TypeScript enforces the correct method:

```typescript
// TypeScript requires method for DELETE/PUT/PATCH
const result = await api.mutate({
	pattern: "/users/:id",
	params: { id: "123" },
	requestInit: { method: "DELETE" }, // Required by compiler
});
```

#### Request Customization

Both queries and mutations accept `requestInit` for customization:

```typescript
const result = await api.query({
	pattern: "/data",
	requestInit: {
		headers: {
			Authorization: "Bearer token",
			"X-Custom": "value",
		},
		credentials: "include",
		signal: abortController.signal, // For cancellation
	},
});
```

#### Deduplication

The `dedupeKey` option prevents duplicate concurrent requests:

```typescript
// Click button multiple times - only one request sent
async function handleSave() {
	const result = await api.mutate({
		pattern: "/save",
		input: data,
		options: {
			dedupeKey: "save-button", // Same key = dedupe
		},
	});
}
```

By default, mutations automatically trigger revalidation of the current route's
data. Set `revalidate: false` to skip this.

### Client Setup

In `frontend/entry.tsx`:

```typescript
import { render } from "preact";
import { initClient, getRootEl } from "river.now/client";
import { App } from "./app.tsx";
import { apiConfig } from "./river.gen.ts";

await initClient(() => render(<App />, getRootEl()), {
  apiConfig,
  defaultErrorBoundary: (props) => <div>Error: {props.error}</div>,
});

// Optional: Loading indicators
import { done, isStarted, start } from "nprogress";
import { setupGlobalLoadingIndicator } from "river.now/client";
setupGlobalLoadingIndicator({ start, stop: done, isRunning: isStarted });
```

## TypeScript Generation

Running `npm run dev` or `npm run build` generates `frontend/river.gen.ts`:

```typescript
// Generated from your Go types
type RootData = {
  Message: string;
  Count: number;
};

// Pattern literals
type RiverLoaderPattern = "/" | "/_index" | "/users/:id";

// Typed I/O based on pattern
type RiverLoaderOutput<"/"> = RootData;
type RiverLoaderOutput<"/users/:id"> = UserData;

// Route params extraction
type RiverRouteParams<"/users/:id"> = "id";
```

## Build System

River's build process:

1. Parses Go types → generates TypeScript
2. Discovers client routes from `routes.ts`
3. Bundles with Vite
4. Manages HMR in development
5. Optimizes for production

Commands:

- `npm run dev` - Development with HMR
- `npm run build` - Production build

## Important Notes

1. **Validation is not automatic** - It only happens if you use River's validate
   helpers in your ParseInput
2. **Loaders are optional** - Client-only routes work fine without server
   loaders
3. **Error boundaries only work with 200 OK** - If you set an error status
   (400+) via ResponseProxy, you get an HTTP error instead
4. **Client loaders run in parallel** - Once discovered, they run alongside
   server loaders
5. **ResponseProxy merging** - First error wins, then first redirect, then last
   success
6. **Middleware execution** - HTTP middleware is sequential, task middleware is
   parallel
