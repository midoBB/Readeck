// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

const safeMethods = ["GET", "HEAD", "OPTIONS", "TRACE"]

const csrfToken = document.querySelector(
  'html>head>meta[name="x-csrf-token"]',
).content

async function request(path, options) {
  const {headers, query = null, method = "GET", body, ...extraOpts} = options

  // Prep options
  const reqOptions = {
    method: method.toUpperCase(),
    headers: new Headers({
      ...(!safeMethods.includes(method) && {"X-CSRF-Token": csrfToken}),
      ...headers,
    }),
  }

  if (body) {
    // Automatic body serialization only when content-type is not set
    // or body is not a FormData
    if (
      typeof body == "object" &&
      !(body instanceof FormData) &&
      !reqOptions.headers.has("content-type")
    ) {
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
