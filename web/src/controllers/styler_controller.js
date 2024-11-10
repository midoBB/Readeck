// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

// Controller "styler" is only an outlet for the "styler-option" controller.
export default class extends Controller {
  initialize() {
    // only add the styler-option controller the first time
    // the controller is initialized.
    if (
      this.application.controllers.find(
        (x) => x.context.controller.identifier == "styler-option",
      ) === undefined
    ) {
      this.application.register("styler-option", stylerOption)
    }
  }
}

class stylerOption extends Controller {
  static outlets = ["styler"]
  static values = {
    current: {
      type: String,
      default: "",
    },
    values: {
      type: Array,
      default: [],
    },
  }
  static targets = ["choices", "value"]

  constructor() {
    super(...arguments)
    this.choices = {}
  }

  connect() {
    this.valueTargets.forEach((e) => {
      this.currentValue = e.value
    })

    this.choicesTargets.forEach((e) => {
      this.choices[e.value] = e.dataset.choiceValue
    })
  }

  stylerOutletConnected() {
    if (this.currentValue === null) {
      return
    }
    this.applyClass()
    this.updateChoices()
  }

  updateChoices() {
    let found = false
    this.choicesTargets.forEach((e) => {
      if (e.value == this.currentValue) {
        e.setAttribute("data-current", "1")
        found = true
      } else {
        e.removeAttribute("data-current")
      }
    })

    if (!found && this.hasChoicesTarget) {
      // Set current to the first choice
      this.choicesTarget.setAttribute("data-current", "1")
    }
  }

  dispatchEvents() {
    this.valueTargets.forEach((e) => this.dispatch("setValue", {target: e}))
  }

  setChoice(evt) {
    this.currentValue = evt.target.value
  }

  increaseValue() {
    const value = parseInt(this.currentValue)
    if (!isNaN(value) && value == this.valuesValue.length) {
      return
    }
    this.currentValue = value + 1
  }

  decreaseValue() {
    const value = parseInt(this.currentValue)
    if (!isNaN(value) && value == 1) {
      return
    }
    this.currentValue = value - 1
  }

  currentValueChanged(value, prev) {
    if (!prev) {
      return
    }
    this.valueTargets.forEach((e) => (e.value = value))
    this.applyClass()
    this.updateChoices()
    this.dispatchEvents()
  }

  getAllClasses() {
    let res = []
    if (this.valuesValue.length > 0) {
      res = this.valuesValue
    } else {
      res = Object.values(this.choices)
    }
    return res.reduce((acc, cur) => {
      return acc.concat(cur.split(/\s+/))
    }, [])
  }

  getCurrentClasses() {
    const idx = parseInt(this.currentValue)
    let res = ""
    if (this.valuesValue.length > 0 && !isNaN(idx)) {
      res = this.valuesValue[idx - 1]
    } else {
      res = this.choices[this.currentValue]
    }

    if (!!res) {
      return res.split(/\s+/)
    }
    return []
  }

  applyClass() {
    const classNames = this.getCurrentClasses()
    if (classNames.length == 0) {
      return
    }

    this.stylerOutletElements.forEach((e) => {
      e.classList.remove(...this.getAllClasses())
      e.classList.add(...classNames)
    })
  }
}
