import { Controller } from "@hotwired/stimulus"

import $ from "../lib/dq"
import icon from "../lib/icon"

export default class extends Controller {
  static get targets() {
    return ["editable", "value", "button"]
  }

  static get values() {
    return {
      saveIcon: {
        type: String,
        default: "o-check-on",
      },
    }
  }

  connect() {
    this.editableTarget.tabIndex = 0
    this.icon = $("span.svgicon", this.buttonTarget).get()

    this.editableTarget.addEventListener("keydown", (evt) => {
      if (evt.key == "Enter") {
        evt.preventDefault()
      }
    })
    this.editableTarget.addEventListener("keyup", (evt) => {
      if (evt.key == "Enter") {
        evt.preventDefault()
        this.buttonTarget.dispatchEvent(new MouseEvent("click"))
      }
    })

    this.editableTarget.addEventListener("focus", () => {
      if (this.editableTarget.contentEditable !== "true") {
        this.startEditable()
      }
    })

    this.editableTarget.addEventListener("click", () => {
      if (this.editableTarget.contentEditable !== "true") {
        this.startEditable()
      }
    })

    this.buttonTarget.addEventListener("click", (evt) => {
      // Enable contenteditable on first click
      if (this.editableTarget.contentEditable !== "true") {
        evt.preventDefault()
        this.startEditable()
        return
      }

      // Submit value otherwise
      this.valueTarget.value = this.editableTarget.textContent
    })
  }

  startEditable() {
    this.editableTarget.contentEditable = "true"
    this.editableTarget.focus()

    if (this.icon !== undefined && this.saveIconValue) {
      icon.swapIcon(this.icon.firstChild, this.saveIconValue)
    }
  }
}
