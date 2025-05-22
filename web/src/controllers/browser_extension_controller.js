// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static classes = ["installed"]
  static values = {
    ready: {
      type: Boolean,
      default: undefined, // start with undefined so we don't trigger any change
    },
  }

  async connect() {
    window.addEventListener("message", (evt) => {
      // The message coming from the extension sets "ready" to true.
      if (evt.origin == window.location.origin) {
        if (evt.data == "readeck-ready") {
          this.readyValue = true
        }
      }
    })

    // In order to switch to not "ready", we must wait for
    // the event to never arrive.
    setTimeout(() => {
      if (this.readyValue === undefined) {
        this.readyValue = false
      }
    }, 300)
  }

  readyValueChanged(value) {
    if (value === false) {
      this.element.classList.remove(...this.installedClasses)
      return
    }
    this.element.classList.add(...this.installedClasses)
  }

  setReady() {
    this.readyValue = true
  }
}
