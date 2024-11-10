// SPDX-FileCopyrightText: © 2023 Joachim Robert <joachim.robert@protonmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["panel", "button"]
  static classes = ["hidden", "body"]

  toggle() {
    if (this.isVisible()) {
      this.close()
    } else {
      this.open()
    }

    this.buttonTarget.setAttribute("aria-expanded", this.isVisible())
  }

  open() {
    document.body.classList.add(...this.bodyClasses)
    this.panelTarget.classList.remove(...this.hiddenClasses)
    this.panelTarget.focus()
  }

  close() {
    document.body.classList.remove(...this.bodyClasses)
    this.panelTarget.classList.add(...this.hiddenClasses)
    this.buttonTarget.focus()
  }

  /**
   * isVisible returns true when the menu/panel is visible
   *
   * @returns boolean
   */
  isVisible() {
    const cs = getComputedStyle(this.panelTarget)
    return !(cs.visibility == "hidden" || cs.display == "none")
  }
}
