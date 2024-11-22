// SPDX-FileCopyrightText: © 2023 Joachim Robert <joachim.robert@protonmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["panel", "button"]
  static classes = ["hidden", "forceHidden", "body"]

  toggle() {
    if (this.isVisible()) {
      this.close()
    } else {
      this.open()
    }
  }

  open() {
    document.body.classList.add(...this.bodyClasses)
    this.panelTarget.classList.remove(...this.hiddenClasses)
    this.panelTarget.focus()
    this.buttonTarget.setAttribute("aria-expanded", this.isVisible())
  }

  close(evt) {
    document.body.classList.remove(...this.bodyClasses)
    if (evt === undefined) {
      this.panelTarget.classList.add(...this.hiddenClasses)
      this.buttonTarget.focus()
    } else {
      this.panelTarget.classList.add(...this.forceHiddenClasses)
    }
    this.buttonTarget.setAttribute("aria-expanded", this.isVisible())
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
