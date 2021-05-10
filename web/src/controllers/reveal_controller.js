import { Controller } from "stimulus"

export default class extends Controller {
  static get targets () {
    return ["trigger", "content"]
  }
  static get values () {
    return {
      visible: Boolean,
    }
  }

  connect () {
    this.triggerTargets.forEach(e => {
      e.style.cursor = "pointer"
      e.setAttribute("data-action", `click->${this.identifier}#toggle`)
    })
  }

  visibleValueChanged() {
    if (this.visibleValue) {
      this.contentTargets.forEach(e => e.style.display = "")
    } else {
      this.contentTargets.forEach(e => e.style.display = "none")
    }
  }

  toggle(evt) {
    if (evt) {
      evt.preventDefault()
    }
    this.visibleValue = !this.visibleValue
  }
}
