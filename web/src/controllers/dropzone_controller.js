// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["zone", "placeholder", "fileinfo", "clearbtn"]
  static classes = ["hidden", "focus"]

  connect() {
    this.actions = []

    if (!this.input) {
      return
    }

    this.input.removeAttribute("required")
    this.actions = [
      [this.input, "change", () => this.setFileInfo()],
      [this.element, "dragenter", (e) => this.enter(e)],
      [this.element, "dragleave", (e) => this.leave(e)],
      [this.element, "drop", (e) => this.drop(e)],
      [document, "dragover", (e) => this.noop(e)],
      [document, "dragleave", (e) => this.noop(e)],
      [document, "drop", (e) => this.noop(e)],
    ]

    this.actions.forEach((a) => {
      a[0].addEventListener(a[1], a[2])
    })
    this.setFileInfo()
  }

  disconnect() {
    this.actions.forEach((a) => {
      a[0].removeEventListener(a[1], a[2])
    })
  }

  /**
   * @returns {HTMLInputElement}
   */
  get input() {
    return this.scope.findElement("input[type=file]")
  }

  setFileInfo() {
    this.zoneTargets.forEach((e) => {
      e.classList.remove(this.focusClass)
    })

    const files = this.input.files
    if (files.length == 1) {
      this.placeholderTargets.forEach((e) => e.classList.add(this.hiddenClass))
      this.clearbtnTargets.forEach((e) => e.classList.remove(this.hiddenClass))

      this.fileinfoTargets.forEach((e) => {
        e.innerText = `${files[0].name} (${formatBytes(files[0].size)})`
      })
      return
    }

    this.placeholderTargets.forEach((e) => e.classList.remove(this.hiddenClass))
    this.clearbtnTargets.forEach((e) => e.classList.add(this.hiddenClass))
    this.fileinfoTargets.forEach((e) => {
      e.innerText = ""
    })
  }

  clear() {
    this.input.value = null
    this.input.dispatchEvent(
      new Event("input", {bubbles: true, cancelable: true}),
    )
    this.input.dispatchEvent(
      new Event("change", {bubbles: true, cancelable: true}),
    )
  }

  select(_) {
    this.input.click()
  }

  /**
   *
   * @param {Event} evt
   */
  enter(evt) {
    evt.preventDefault()
    this.zoneTargets.forEach((e) => {
      e.classList.add(this.focusClass)
    })
  }

  /**
   *
   * @param {Event} evt
   */
  leave(evt) {
    evt.preventDefault()
    this.zoneTargets.forEach((e) => {
      if (!e.contains(evt.relatedTarget)) {
        e.classList.remove(this.focusClass)
      }
    })
  }

  /**
   *
   * @param {Event} evt
   */
  drop(evt) {
    evt.preventDefault()
    if (evt.dataTransfer.files.length != 1) {
      return
    }
    let input = this.input
    if (input) {
      input.files = evt.dataTransfer.files
      input.dispatchEvent(new Event("input", {bubbles: true, cancelable: true}))
      input.dispatchEvent(
        new Event("change", {bubbles: true, cancelable: true}),
      )
    }
  }

  /**
   *
   * @param {Event} evt
   */
  noop(evt) {
    evt.preventDefault()
  }
}

function formatBytes(bytes, decimals = 2) {
  if (!+bytes) return "0 Bytes"

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = [
    "Bytes",
    "KiB",
    "MiB",
    "GiB",
    "TiB",
    "PiB",
    "EiB",
    "ZiB",
    "YiB",
  ]

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}
