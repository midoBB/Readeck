// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  // static targets = ["close"]
  static values = {
    delay: {
      type: Number,
      default: 0,
    },
  }

  connect() {
    if (this.delayValue > 0) {
      window.setTimeout(() => this.remove(), this.delayValue * 1000)
    }
  }

  remove() {
    this.element.addEventListener("transitionend", () => this.element.remove())
    this.element.style.opacity = 0
  }
}
