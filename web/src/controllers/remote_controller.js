// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  toggleDetails(evt) {
    if (!evt.params.selector) {
      return
    }
    document.querySelectorAll(evt.params.selector).forEach((e) => {
      e.open = !e.open
    })
  }
}
