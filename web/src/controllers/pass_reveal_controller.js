// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["field", "btn", "show", "hide"]

  connect() {
    this.btnTarget.appendChild(this.showTarget.content.cloneNode(true))
  }

  toggle() {
    this.btnTarget.replaceChildren()
    if (this.fieldTarget.type == "password") {
      this.btnTarget.appendChild(this.hideTarget.content.cloneNode(true))
      this.fieldTarget.type = "text"
    } else {
      this.btnTarget.appendChild(this.showTarget.content.cloneNode(true))
      this.fieldTarget.type = "password"
    }
    this.fieldTarget.focus()
  }
}
