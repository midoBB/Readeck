// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  async back(evt) {
    evt.preventDefault()
    history.back()

    setTimeout(() => {
      // this is called only when we can't go back in history.
      window.location.href = this.element.getAttribute("href")
    }, 200)
  }
}
