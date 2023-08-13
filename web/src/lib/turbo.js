// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import * as Turbo from "@hotwired/turbo"
import {safeMethods, csrfToken} from "./request"

document.addEventListener("turbo:before-fetch-request", (evt) => {
  // Insert the CSRF token when needed
  let meth = evt.detail.fetchOptions.method.toUpperCase()
  if (!safeMethods.includes(meth)) {
    evt.detail.fetchOptions.headers["X-CSRF-Token"] = csrfToken
  }

  // Mark the request for turbo rendering
  evt.detail.fetchOptions.headers["X-Turbo"] = "1"
})

document.addEventListener("turbo:submit-end", (evt) => {
  // Empty children with data-turbo-empty-submit-end
  // attribute after form submission.
  evt.target
    .querySelectorAll("[data-turbo-empty-submit-end]")
    .forEach((x) => (x.value = ""))
})
