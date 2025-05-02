---
title: FAQs
description: Frequently asked questions about the river.now framework
---

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
