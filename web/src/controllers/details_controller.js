// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["closer"]

  connect() {
    if (this.element.tagName.toLowerCase() != "details") {
      return
    }

    this.element.addEventListener("toggle", () => {
      if (this.element.open) {
        document.addEventListener("click", this.closeDetails)
        document.addEventListener("keyup", this.closeDetails)
      } else {
        document.removeEventListener("click", this.closeDetails)
        document.removeEventListener("keyup", this.closeDetails)
      }
    })

    this.closerTargets.forEach((e) => {
      e.addEventListener("click", () => {
        this.element.open = false
      })
    })
  }

  closeDetails = (evt) => {
    if (evt.type == "keyup") {
      if (evt.key == "Escape") {
        this.element.open = false
      }
      return
    }
    if (!this.element.contains(evt.target)) {
      this.element.open = false
    }
  }
}
