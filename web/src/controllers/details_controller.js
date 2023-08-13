// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  connect() {
    if (this.element.tagName.toLowerCase() != "details") {
      return
    }

    this.element.addEventListener("toggle", () => {
      if (this.element.open) {
        document.addEventListener("click", this.closeDetails)
      } else {
        document.removeEventListener("click", this.closeDetails)
      }
    })
  }

  closeDetails = (evt) => {
    if (!this.element.contains(evt.target)) {
      this.element.open = false
    }
  }
}
