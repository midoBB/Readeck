// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["progress"]

  connect() {
    this.updateProgress()
    let ticking = false

    document.addEventListener("scroll", () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          this.updateProgress()
          ticking = false
        })

        ticking = true
      }
    })
  }

  updateProgress() {
    const p = document.body.parentNode
    const pct = (
      (document.body.scrollTop || p.scrollTop) /
      (p.scrollHeight - p.clientHeight)
    ).toFixed(2)
    this.progressTarget.style.width = `${pct * 100}%`
  }
}
