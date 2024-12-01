// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"
import {request} from "../lib/request"

export default class extends Controller {
  static values = {
    url: {type: String, default: ""},
    method: {type: String, default: ""},
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
    }

    // Set the body based on the form elements when the event receiver
    // is a form.
    if (event.currentTarget.tagName.toLowerCase() == "form") {
      const form = event.currentTarget
      options.body = {}
      for (let e of form.elements) {
        if (!!e.name) {
          let value = e.value
          if (!isNaN(value) && value !== "") {
            value = Number(value)
          }
          options.body[e.name] = value
        }
      }
    }

    await request(src, options)
    this.dispatch(event.params.eventName || "done")
  }
}
