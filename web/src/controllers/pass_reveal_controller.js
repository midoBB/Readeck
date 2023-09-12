// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["field", "template", "show", "hide"]
  static classes = ["hidden"]

  showPassword() {
    this.fieldTarget.type = "text"
    this.showTarget.classList.add(this.hiddenClass)
    this.hideTarget.classList.remove(this.hiddenClass)
    this.fieldTarget.focus()
  }

  hidePassword() {
    this.fieldTarget.type = "password"
    this.showTarget.classList.remove(this.hiddenClass)
    this.hideTarget.classList.add(this.hiddenClass)
    this.fieldTarget.focus()
  }
}
