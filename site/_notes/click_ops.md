# Click Ops

## railway.com

- Set "Root Directory" to "/"
- Set "Railway Config File" to "/site/railway.json"
- Set up a custom domain for "river.now" (naked, without www)

## cloudflare.com

- Set up a CNAME (not A) for both @ and www, with the naked @ pointing to the
  provided railway URL, and the www pointing to the literal naked domain
  ("river.now") (in "DNS" section)
- Set SSL/TLS mode to "Full (strict)" (in "SSL/TLS" section)
- Set up a "Redirect from HTTP to HTTPS" redirect rule from the template, BUT
  check the box to preserve the full query string (in "Rules" section)
- Set up a "Redirect from WWW to Root" redirect rule from the template, BUT
  check the box to preserve the full query string (in "Rules" section)
- Check the box to "Add security headers" (in "Rules" settings)
- Turn off websockets and IP geolocation (in "Network" settings)
- Add a "Cache Rule" to respect origin:
  - All incoming requests
  - Eligible for cache
  - Edge TTL: use cache-control header, else bypass
  - Browser TTL: respect origin TTL
  - Cache key: sort query string
  - Use strong ETag headers
