// SPDX-FileCopyrightText: © 2023 Joachim Robert <joachim.robert@protonmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["panel", "button"]
  static classes = ["hidden", "body"]

  toggleMenu() {
    if (this.isVisible()) {
      document.body.classList.remove(...this.bodyClasses)
      this.panelTarget.classList.add(...this.hiddenClasses)
      this.buttonTarget.focus()
    } else {
      document.body.classList.add(...this.bodyClasses)
      this.panelTarget.classList.remove(...this.hiddenClasses)
      this.panelTarget.focus()
    }

    this.buttonTarget.setAttribute("aria-expanded", this.isSidemenuOn)
  }

  /**
   * isVisible returns true when the menu/panel is visible
   *
   * @returns boolean
   */
  isVisible() {
    return !this.panelTarget.classList.contains(this.hiddenClass)
  }
}
