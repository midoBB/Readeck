// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["embed", "play"]

  connect() {
    this.ifr = this.embedTarget.content.querySelector("iframe")
    if (!this.ifr) {
      return
    }

    this.ifr.setAttribute("sandbox", "allow-scripts allow-same-origin")

    const w = parseInt(this.ifr.getAttribute("width")) || 0
    const h = parseInt(this.ifr.getAttribute("height")) || 0

    if (w > 0 && h > 0) {
      this.element.style.paddingTop = `${(100 * h) / w}%`
    }

    this.playBtn = this.playTarget.content.querySelector("div")
    this.element.appendChild(this.playBtn)
  }

  play() {
    this.playBtn.remove()
    this.element.appendChild(this.ifr)
  }
}
