// SPDX-FileCopyrightText: © 2022 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["editable", "value", "button", "iconOff", "iconOn"]
  static classes = ["hidden"]

  connect() {
    this.editableTarget.tabIndex = 0
    this.editableTarget.setAttribute("spellcheck", "false")

    this.editableTarget.addEventListener("keydown", this.onKeyDown)
    this.editableTarget.addEventListener("keyup", this.onKeyUp)

    this.editableTarget.addEventListener("click", this.onFocus)
    this.editableTarget.addEventListener("focus", this.onFocus)

    this.buttonTarget.addEventListener("click", this.onButtonClick)
  }

  startEditable() {
    this.editableTarget.contentEditable = "true"
    this.editableTarget.focus()
    this.iconOffTarget.classList.add(this.hiddenClass)
    this.iconOnTarget.classList.remove(this.hiddenClass)
  }

  stopEditable() {
    this.editableTarget.contentEditable = "false"
    this.editableTarget.textContent = this.valueTarget.value
    this.iconOffTarget.classList.remove(this.hiddenClass)
    this.iconOnTarget.classList.add(this.hiddenClass)
  }

  onKeyUp = (evt) => {
    switch (evt.key) {
      case "Enter":
        evt.preventDefault()
        this.buttonTarget.dispatchEvent(new MouseEvent("click"))
        break
      case "Escape":
        this.stopEditable()
        break
    }
  }

  onKeyDown = (evt) => {
    if (evt.key == "Enter") {
      evt.preventDefault()
    }
  }

  onFocus = (evt) => {
    if (this.editableTarget.contentEditable !== "true") {
      this.startEditable()
    }
  }

  onButtonClick = (evt) => {
    // Enable contenteditable on first click
    if (this.editableTarget.contentEditable !== "true") {
      evt.preventDefault()
      this.startEditable()
      return
    }

    // Submit value otherwise
    this.valueTarget.value = this.editableTarget.textContent
  }
}
