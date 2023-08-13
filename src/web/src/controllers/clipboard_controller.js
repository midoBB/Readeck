// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"
import $ from "../lib/dq"
import icon from "../lib/icon"

export default class extends Controller {
  static targets = ["label", "content"]

  connect() {
    if (this.hasLabelTarget) {
      $.E("button")
        .addClass("text-primary", "hf:text-primary-dark")
        .attr("type", "button")
        .attr("data-action", `${this.identifier}#copy`)
        .append(icon.getIcon("o-copy"))
        .appendTo(this.labelTarget)
    }
  }

  async copy() {
    await navigator.clipboard.writeText(this.contentTarget.value)
  }
}
