import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  initialize() {
    this.application.register("reader-option", readerOption)
  }
}

class readerOption extends Controller {
  static outlets = ["reader"]
  static values = {
    current: String,
    choices: {
      type: Object,
      default: {},
    },
    values: {
      type: Array,
      default: [],
    },
  }
  static targets = ["control", "increase", "decrease", "value"]

  readerOutletConnected() {
    this.applyClass()
    this.updateControls()
  }

  controlTargetConnected(el) {
    el.addEventListener("click", (evt) => {
      this.setValue(el.getAttribute("value"))
    })
  }

  increaseTargetConnected(el) {
    if (this.valuesValue.length == 0) {
      return
    }
    el.addEventListener("click", () => this.increaseValue())
  }

  decreaseTargetConnected(el) {
    if (this.valuesValue.length == 0) {
      return
    }
    el.addEventListener("click", () => this.decreaseValue())
  }

  updateControls() {
    // update control target position
    this.controlTargets.forEach((el) => {
      if (el.getAttribute("value") == this.currentValue) {
        el.setAttribute("data-current", "1")
        return
      }
      el.removeAttribute("data-current")
    })

    if (this.valuesValue.length == 0) {
      return
    }
    const value = parseInt(this.currentValue)

    // toggle increase and decrease
    this.valueTargets.forEach((e) => (e.value = value))
    this.decreaseTargets.forEach((e) => (e.disabled = value == 1))
    this.increaseTargets.forEach((e) => {
      e.disabled = value >= this.valuesValue.length
    })
  }

  setValue(value) {
    this.currentValue = value
  }

  increaseValue() {
    const value = parseInt(this.currentValue)
    if (value == this.valuesValue.length) {
      return
    }
    this.currentValue = value + 1
  }

  decreaseValue() {
    const value = parseInt(this.currentValue)
    if (value == 1) {
      return
    }
    this.currentValue = value - 1
  }

  currentValueChanged(value, prev) {
    if (!prev) {
      return
    }
    this.applyClass()
    this.updateControls()
  }

  getAllClasses() {
    if (this.valuesValue.length > 0) {
      return this.valuesValue
    }
    return Object.values(this.choicesValue)
  }

  getCurrentClass() {
    const idx = parseInt(this.currentValue)
    if (this.valuesValue.length > 0 && !isNaN(idx)) {
      return this.valuesValue[idx - 1]
    }
    return this.choicesValue[this.currentValue]
  }

  applyClass() {
    const className = this.getCurrentClass()
    if (!className) {
      return
    }

    this.readerOutletElement.classList.remove(...this.getAllClasses())
    this.readerOutletElement.classList.add(className)
  }
}
