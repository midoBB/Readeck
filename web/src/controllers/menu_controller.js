// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static values = {
    offset: String,
  }

  connect() {
    if (this.element.tagName.toLowerCase() != "details") {
      return
    }

    this.positionMenu()
    window.addEventListener("resize", () => {
      this.positionMenu()
    })

    this.element.addEventListener("toggle", () => {
      if (this.element.open) {
        document.addEventListener("click", this.closeDetails)
        document.addEventListener("keyup", this.closeDetails)
      } else {
        document.removeEventListener("click", this.closeDetails)
        document.removeEventListener("keyup", this.closeDetails)
      }
    })
  }

  toggle = () => {
    this.element.open = !this.element.open
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

  positionMenu = () => {
    if (!this.offsetValue) {
      return
    }

    const el = this.element.querySelector("ul:first-of-type")
    if (!el) {
      return
    }
    const styleName = this.offsetValue
    const positionName = styleName == "left" ? "right" : "left"

    if (getComputedStyle(el).position != "absolute") {
      return
    }

    // Remove the style first, so the classes have priority
    el.style[styleName] = ""

    const bound = el.getBoundingClientRect()[positionName]
    const ww = window.innerWidth
    if (styleName == "right") {
      el.style[styleName] = bound < 0 ? `${bound - 4}px` : "0"
    } else {
      el.style[styleName] = bound > ww ? `${ww - bound - 4}px` : "0"
    }
  }
}
