// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"
import {request} from "../lib/request"
import {cspNonce} from "../lib/turbo"

export default class extends Controller {
  static values = {
    url: {type: String, default: ""},
    method: {type: String, default: ""},
    turbo: {type: Boolean, default: false},
  }

  async fetch(event) {
    const src = !!event.params.url ? event.params.url : this.urlValue
    if (!src) {
      throw new Error("url param is not set")
    }

    const method = !!event.params.method
      ? event.params.method
      : this.methodValue
    const options = {
      method: method || "get",
      headers: {},
    }

    if (this.turboValue) {
      options.headers["X-Turbo"] = "1"
      options.headers = {
        ...options.headers,
        "X-Turbo": "1",
        "X-Turbo-Nonce": cspNonce,
      }
    }

    if (event.currentTarget.tagName.toLowerCase() == "form") {
      // Set the body based on the form elements when the event receiver
      // is a form.
      const form = event.currentTarget
      options.body = {}
      for (let e of form.elements) {
        if (!!e.name) {
          options.body[e.name] = getValue(e.value)
        }
      }
    } else if (
      event.currentTarget.name !== undefined &&
      event.currentTarget.value !== undefined
    ) {
      // Otherwise, anything with a name and value attribute does the job.
      options.body = {}
      options.body[event.currentTarget.name] = getValue(
        event.currentTarget.value,
      )
    }

    const rsp = await request(src, options)
    if (this.turboValue) {
      Turbo.renderStreamMessage(await rsp.text())
    }

    this.dispatch(event.params.eventName || "done")
  }
}

function getValue(value) {
  if (!isNaN(value) && value !== "") {
    return Number(value)
  }
  return value
}
