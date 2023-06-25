const safeMethods = ["GET", "HEAD", "OPTIONS", "TRACE"]

const csrfToken = document.querySelector(
  'html>head>meta[name="x-csrf-token"]',
).content

async function request(path, options) {
  const {headers, query = null, method = "GET", body, ...extraOpts} = options

  // Prep options
  const reqOptions = {
    method,
    headers: new Headers({
      ...(!safeMethods.includes(method) && {"X-CSRF-Token": csrfToken}),
      ...headers,
    }),
  }

  if (body) {
    // Automatic body serialization only when content-type is not set
    if (typeof body == "object" && !reqOptions.headers.has("content-type")) {
      reqOptions.body = JSON.stringify(body)
      reqOptions.headers.set("Content-Type", "application/json")
    } else {
      reqOptions.body = body
    }
  }

  // Prep URL
  let qs = ""
  if (query) {
    qs = new URLSearchParams(query).toString()
    qs = qs && `?${qs}`
  }

  const req = new Request(`${path}${qs}`, reqOptions)
  return await fetch(req)
}

export {safeMethods, csrfToken, request}
