// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["icon", "iconLight", "iconDark", "iconSystem"]

  connect() {
    this.setIcon()
    window.addEventListener("storage", () => {
      this.setTheme()
      this.setIcon()
    })
  }

  get currentTheme() {
    if (localStorage.getItem("theme") !== null) {
      return localStorage.theme
    }
    return "system"
  }

  toggleTheme() {
    switch (this.currentTheme) {
      case "system":
        localStorage.theme = "dark"
        break
      case "dark":
        localStorage.theme = "light"
        break
      case "light":
        localStorage.removeItem("theme")
        break
    }
    window.dispatchEvent(new Event("storage"))
  }

  setTheme() {
    let theme = "light"
    switch (this.currentTheme) {
      case "light":
        break
      case "dark":
        theme = "dark"
        break
      case "system":
        if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
          theme = "dark"
        }
        break
    }

    if (theme == "dark") {
      document.documentElement.classList.add("dark")
    } else {
      document.documentElement.classList.remove("dark")
    }
  }

  changeIcon(tpl) {
    this.iconTarget.replaceChildren(tpl.content.cloneNode(true))
  }

  setIcon() {
    switch (this.currentTheme) {
      case "light":
        this.changeIcon(this.iconLightTarget)
        break
      case "dark":
        this.changeIcon(this.iconDarkTarget)
        break
      case "system":
        this.changeIcon(this.iconSystemTarget)
        break
    }
  }
}
