# River Client Navigation: Complete Behavior Specification

This document provides a comprehensive specification of the River client-side
navigation system's behavior, suitable for testing and refactoring.

## 1. Core Navigation

### 1.1. Navigation Types

The system uses five distinct navigation types:

- **`userNavigation`**: Triggered by user clicking a link or calling
  `navigate()`. Primary navigation type.
- **`browserHistory`**: Triggered by browser back/forward buttons (POP events).
- **`revalidation`**: Background refetch of current page data via `revalidate()`
  or after non-GET submissions. Does not affect history or scroll.
- **`redirect`**: Client-side redirect from server response or submission
  result.
- **`prefetch`**: Speculative low-priority fetch, typically on link hover.

### 1.2. Navigation State Management

- **Single Active User Navigation**: Only one `userNavigation` can be active.
  Starting a new one aborts ALL other navigations (including prefetches).
- **Navigation Tracking**: All active navigations stored in
  `navigationState.navigations` Map, keyed by target `href`.
- **Cleanup**: Navigations removed from map when their fetch promise
  resolves/rejects.
- **Active User Navigation Tracking**: `navigationState.activeUserNavigation`
  tracks current user navigation href.

### 1.3. Link Click Handling

- **Internal Links**: Eligible internal links (`<a>` tags, same origin, no
  modifier keys) prevent default and trigger `userNavigation`.
- **External/Special Links**: External domains, `mailto:`, `tel:`, etc.
  ignored - browser handles normally.
- **Modifier Keys**: Cmd/Ctrl/Shift/Alt clicks ignored for browser default
  behavior.
- **Hash-Only Links**:
  - Calls `saveScrollState()` to save current position
  - Allows browser default scroll-to-element behavior
  - Does NOT trigger navigation
- **Already Prefetching**: If link already has completed prefetch data, uses it
  immediately on click.

### 1.4. Programmatic Navigation

- **`navigate(href, options?)`**: Public API for navigation
  - `options.replace`: Uses `history.replaceState` instead of `pushState`
- **`__navigate(props)`**: Internal function taking full NavigateProps

## 2. Navigation Lifecycle

### 2.1. Phase 1: Begin Navigation (`beginNavigation`)

1. **Set Loading Status**: Sets appropriate loading state to `true` (except for
   `prefetch`)
2. **Handle User Navigation Specifics**:
   - Abort all other navigations via `abortAllNavigationsExcept(href)`
   - Set `activeUserNavigation = href`
   - Check for existing prefetch to upgrade (changes type, reuses control)
3. **Prefetch Deduplication**: If prefetch exists for href, return existing
   control
4. **Create Navigation**:
   - New `AbortController`
   - Create control object with promise from `__fetchRouteData`
   - Store in `navigationState.navigations` with type

### 2.2. Phase 2: Fetch Route Data (`__fetchRouteData`)

1. **URL Construction**:
   - Add `river_json=<buildID>` query param (or "1" if no buildID)
   - Use absolute URL from `window.location.href`
2. **Request Execution**:
   - Headers include `X-Accepts-Client-Redirect: 1`
   - Request goes through `handleRedirects` function
3. **Response Validation**:
   - Check for redirect completion or non-OK status (except 304)
   - Empty JSON response is failure
   - Set loading status to false on failure
4. **Asset Preloading**:
   - **Modules**: Create `<link rel="modulepreload">` for each URL in:
     - Production: `json.deps`
     - Development: `json.importURLs` (deduped with `new Set()`)
   - **CSS**: Create `<link rel="preload" as="style">` for each
     `json.cssBundles`
     - Create Promise for each with onload/onerror handlers
     - Store promises in `cssBundlePromises` array
5. **Client Wait Functions**: Execute `runWaitFns(json, buildID)` returning
   promise
6. **Cleanup**: Always remove from `navigations` map and clear
   `activeUserNavigation` if matches

### 2.3. Phase 3: Complete Navigation (`__completeNavigation`)

1. **Redirect Check**: If result has `redirectData`, execute it and return
2. **Build ID Check**:
   - Compare response buildID with current
   - If different, dispatch `river:build-id` event with old/new IDs
3. **Wait for Client Data**: Await `waitFnPromise` from Phase 2
4. **Store Client Data**: Set `clientLoadersData` in global state
5. **Render**: Call `__reRenderApp` with all data

### 2.4. Phase 4: Re-render App (`__reRenderApp`)

1. **Clear Loading State**: Set loading status to false
2. **View Transitions Check**:
   - Only if `useViewTransitions` enabled AND `document.startViewTransition`
     exists
   - Skip for `prefetch` and `revalidation` types
   - Wrap rest of rendering in transition
3. **Update Global State** (`__reRenderAppInner`):
   - Set all route data in `internal_RiverClientGlobal`
   - Keys: `outermostError`, `outermostErrorIdx`, `errorExportKey`,
     `matchedPatterns`, `loadersData`, `importURLs`, `exportKeys`,
     `hasRootData`, `params`, `splatValues`
4. **Load Components**: Call `handleComponents(json.importURLs)`
5. **History Management** (for `userNavigation` and `redirect`):
   - Compare target URL with current
   - Use `push` if different and not replace mode
   - Use `replace` if same URL or replace mode
   - Set scroll state: hash if present, else `{x: 0, y: 0}`
6. **Browser History Scroll** (for `browserHistory`):
   - Use provided `scrollStateToRestore`
   - Or use hash if present
7. **Update Title**:
   - Create temporary `<textarea>` to decode HTML entities
   - Set `document.title` if different
8. **Wait for CSS**:
   - Log "Waiting for CSS bundle preloads..."
   - `await Promise.all(cssBundlePromises)`
   - Log completion or errors
9. **Dispatch Route Change**: Fire `river:route-change` event with scroll state
10. **Apply CSS** (in `requestAnimationFrame`):
    - Check for existing `link[data-river-css-bundle="..."]`
    - Create `<link rel="stylesheet">` for new bundles
    - Set `data-river-css-bundle` attribute for deduplication
11. **Update Head Elements**: Call `updateHeadEls` for meta and other tags

## 3. Prefetching

### 3.1. Initialization (`getPrefetchHandlers`)

- **Eligibility**: Only HTTP, relative, internal URLs
- **Returns**: Object with `start`, `stop`, `onClick` handlers plus href details
- **Already on Page Check**: Compares URLs without hash - won't prefetch current
  page

### 3.2. Prefetch Lifecycle

- **Start**: On `mouseenter`, sets timeout for `delayMs` (default 100ms)
- **Prefetch Execution**:
  - Checks not already prefetching
  - Calls `beforeBegin` callback
  - Starts navigation with type `prefetch`
  - Stores promise result for potential reuse
- **Cancellation** (`stop`):
  - Clears timeout
  - Aborts ONLY if still type `prefetch` (not upgraded)
  - Clears stored nav and result
- **Click with Prefetch**:
  - If prefetch completed, uses stored result immediately
  - Prevents default, shows loading state
  - Upgrades prefetch to `userNavigation`
  - Executes callbacks: `beforeBegin`, `beforeRender`, `afterRender`

## 4. Scroll Restoration

### 4.1. Storage Mechanism

- **SessionStorage Key**: `__river__scrollStateMap`
- **Format**: Map serialized as JSON array of entries
- **Entry Limit**: 50 entries max, oldest evicted on overflow
- **Manual Mode**: Sets `history.scrollRestoration = 'manual'` on init

### 4.2. Saving Scroll State

- **Timing**: Before navigation or on POP to different document
- **Data**: Current `{x: window.scrollX, y: window.scrollY}`
- **Key**: Uses `lastKnownCustomLocation.key` from history

### 4.3. Restoring Scroll State

- **On POP Navigation**:
  - Hash additions/updates: Scroll to element with ID
  - Hash removal: Restore saved position or `{x: 0, y: 0}`
  - Different document: Use saved position for history key
- **Standard Navigation**:
  - With hash: Scroll to element
  - Without: Scroll to top `{x: 0, y: 0}`
- **No State + Hash**: Falls back to
  `document.getElementById(hash)?.scrollIntoView()`

### 4.4. Page Refresh Handling

- **Save on Unload**: Stores position with timestamp and href in
  `__river__pageRefreshScrollState`
- **Restore Check**: On init, if same URL and within 5 seconds, restores
  position
- **Cleanup**: Removes from storage after restore
- **Timing**: Uses `requestAnimationFrame` for restore

## 5. Redirects

### 5.1. Request Configuration

All navigation requests include `X-Accepts-Client-Redirect: 1` header to enable
server-controlled client-side redirects.

### 5.2. Response Headers (Checked in Order)

1. **`X-River-Reload`** (Highest Priority):

   - Forces hard reload to specified URL
   - Always uses hard redirect strategy
   - Ignores other redirect mechanisms

2. **Native Browser Redirect** (`response.redirected`):

   - Only handled for GET requests (non-GET returns null)
   - If redirected to current URL: Returns "did" status (already completed)
   - Otherwise: Soft redirect for internal URLs, hard for external

3. **`X-Client-Redirect`** (Lowest Priority):
   - Custom client-side redirect instruction
   - Soft redirect for internal URLs, hard for external
   - Only checked if no other redirect mechanism triggered

### 5.3. Build ID Tracking

- **Header**: `X-River-Build-Id` on all responses
- **Storage**: Updates global state before redirect execution
- **Events**: Triggers `river:build-id` event if changed

### 5.4. Redirect Strategies

- **Soft Redirect** (Internal URLs):
  - Triggers new navigation with type `redirect`
  - No page reload
  - Preserves SPA experience
- **Hard Redirect** (External URLs or Forced):
  - External: Sets `window.location.href` directly
  - Internal forced: Adds `?river_reload=<buildID>` param and reloads
  - Returns completion data with status "did"

### 5.5. Error Handling

- **Non-HTTP URLs**: Silently ignored (returns null)
- **GET Request Network Failure**: Falls back to `window.location.href = url`
  (except prefetch)
- **Non-GET Redirects**: Logged and ignored

### 5.6. URL Cleanup

On `initClient`, removes `river_reload` param from URL via `history.replace` if
present.

## 6. Form Submissions

### 6.1. Submit Function (`submit`)

- **Deduplication**: Same URL+method aborts previous submission
- **Loading State**: Sets `isSubmitting = true`
- **Request Handling**:
  - FormData and strings sent as-is
  - Other bodies JSON stringified
  - Goes through redirect handling
- **Revalidation**:
  - Non-GET: Auto-revalidates unless redirected
  - GET: No auto-revalidation
  - Manages loading state transition (submission → revalidation)
- **Returns**: `{success: true, data: T}` or `{success: false, error: string}`

### 6.2. Revalidate Function

- **Debouncing**: 10ms debounce via `debounce` utility
- **Type**: Uses `navigationType: "revalidation"`
- **Target**: Current `window.location.href`

## 7. Events System

### 7.1. Loading States (`river:status`)

- **States**: `isNavigating`, `isSubmitting`, `isRevalidating`
- **Debouncing**: 5ms to prevent flicker
- **Deduplication**: Uses `jsonDeepEquals` to prevent duplicate events
- **Access**: `getStatus()` returns current state synchronously

### 7.2. Route Changes (`river:route-change`)

- **Timing**: After navigation completes, before UI updates
- **Detail**: Contains `scrollState` and optional `index`
- **Listeners**: Added via `addRouteChangeListener`

### 7.3. Location Changes (`river:location`)

- **Trigger**: When `location.key` changes in history
- **Purpose**: Signals new history entry created

### 7.4. Build ID Changes (`river:build-id`)

- **Detail**: `{oldID, newID, fromGETAction}`
- **Storage**: Updates global `buildID` before dispatching

## 8. Component & Module Loading

### 8.1. Initial Load

- Uses `importURLs` array to dynamically import modules
- Maps modules using `exportKeys` array (defaults to "default")
- Stores in `activeComponents` array

### 8.2. Error Boundaries

- Uses `outermostErrorIdx` to find error component
- Falls back to `defaultErrorBoundary` if not found
- Uses `errorExportKey` for non-default exports

### 8.3. URL Resolution

- Development: Adds `?river_dev=1` to all imports
- Uses `viteDevURL` in dev, `publicPathPrefix` in prod
- Handles trailing slashes correctly

## 9. Development Features

### 9.1. HMR Support

- **Setup**: `hmrRunClientLoaders` tracks files with HMR
- **Updates**: Listens for `vite:afterUpdate` events
- **Revalidation**: Debounced by 10ms when relevant files change
- **Global**: `window.__waveRevalidate` available for debugging

### 9.2. HMR State

- Tracks `latestHMRTimestamp`
- Logs "HMR update detected" messages
- Only registers HMR once per file via Set

## 10. Initialization (`initClient`)

1. **HMR Setup** (dev only): Register update listener
2. **Configure Options**:
   - `defaultErrorBoundary`: Fallback error component
   - `useViewTransitions`: Enable view transitions
3. **Initialize History**:
   - Create browser history instance
   - Set up POP listener
   - Set `scrollRestoration = 'manual'`
4. **Clean URL**: Remove `river_reload` param if present
5. **Load Initial Components**: Via `handleComponents`
6. **Setup Client Loaders**: Run initial wait functions
7. **Execute Render**: Call user's `renderFn`
8. **Restore Scroll**: Check for post-refresh scroll state
9. **Touch Detection**: First touch sets `isTouchDevice` flag

## 11. History Management

### 11.1. Custom History

- Uses `history` package's `createBrowserHistory`
- Maintains `lastKnownCustomLocation` for comparison
- Listens for all history changes

### 11.2. POP Event Handling

- **Location Key Change**: Dispatches location event
- **Same Document Changes**: Hash-only updates
- **Different Document**: Triggers full navigation
- **Scroll Saving**: Before navigating away

## 12. Error Handling

### 12.1. Abort Errors

- Identified by `isAbortError` utility
- Silently ignored (expected behavior)
- Don't affect loading states

### 12.2. Navigation Failures

- Logged via `LogError`
- Loading state cleared
- User remains on current page
- No partial state updates

### 12.3. Special Cases

- **Empty JSON**: Treated as navigation failure
- **Network Errors**: Logged, navigation cancelled
- **404/500**: Same as network errors

## 13. Utility Functions

### 13.1. Listener Management

- `makeListenerAdder`: Creates typed event listener helpers
- Returns cleanup function for removing listener
- All listeners use `window` as event target

### 13.2. Public Utilities

- `getRootEl()`: Returns `#river-root` element
- `applyScrollState()`: Handles scroll restoration
- `getLocation()`: Returns current pathname, search, hash
- `getBuildID()`: Returns current build ID

This specification captures the complete behavior of the River navigation
system, including all edge cases, timing details, and development-specific
features.

### Addendum: Required Public Exports

export { addBuildIDListener, addLocationListener, addRouteChangeListener,
addStatusListener, applyScrollState, getBuildID, getHistoryInstance,
getLocation, getPrefetchHandlers, getRootEl, getStatus, hmrRunClientLoaders,
initClient, makeLinkOnClickFn, navigate, type RouteChangeEvent, revalidate, type
StatusEvent, submit };
