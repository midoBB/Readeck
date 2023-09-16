// SPDX-FileCopyrightText: © 2023 Joachim Robert <joachim.robert@protonmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["sidemenu", "button"]

  initialize() {
    this.isSidemenuOn = false
  }

  toggleMenu(event) {
    if (this.isSidemenuOn) {
      document.body.classList.remove("overflow-hidden")
      this.sidemenuTarget.classList.add("sidemenu--hidden")
      this.buttonTarget.focus()
    } else {
      document.body.classList.add("overflow-hidden")
      this.sidemenuTarget.classList.remove("sidemenu--hidden")
      this.sidemenuTarget.focus()
    }

    this.isSidemenuOn = !this.isSidemenuOn
    this.buttonTarget.setAttribute("aria-expanded", this.isSidemenuOn)
  }
}
