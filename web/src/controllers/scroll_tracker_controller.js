// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static classes = ["down"]

  connect() {
    let prevScroll = window.scrollY
    let isDown = false
    let ticking = false
    window.addEventListener("scroll", () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          ticking = false
          isDown = window.scrollY > prevScroll
          prevScroll = window.scrollY
          if (isDown) {
            this.element.classList.add(this.downClass)
          } else {
            this.element.classList.remove(this.downClass)
          }
        })

        ticking = true
      }
    })
  }
}
