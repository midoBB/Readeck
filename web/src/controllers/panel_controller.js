// SPDX-FileCopyrightText: © 2023 Joachim Robert <joachim.robert@protonmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["panel", "button"]

  initialize() {
    this.isPanelOn = false
  }

  toggleMenu(event) {
    if (this.isPanelOn) {
      this.panelTarget.classList.add("panel--hidden")
      this.buttonTarget.focus()
    } else {
      this.panelTarget.classList.remove("panel--hidden")
      this.panelTarget.focus()
    }

    this.isPanelOn = !this.isPanelOn
    this.buttonTarget.setAttribute("aria-expanded", this.isPanelOn)
  }
}
