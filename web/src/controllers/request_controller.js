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

    // Set the body based on the target value if it exists, and the
    // provided data-request-name-param attribute
    if (!!event.params.name && event.currentTarget.value !== undefined) {
      options.body = {}
      let value = event.currentTarget.value
      if (!isNaN(value)) {
        value = Number(value)
      }
      options.body[event.params.name] = value
    }

    await request(src, options)
    this.dispatch(event.params.eventName || "done")
  }
}
