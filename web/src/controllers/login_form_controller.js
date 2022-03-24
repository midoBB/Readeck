import {Controller} from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["username", "password"]

  connect() {
    this.usernameTarget.focus()
  }

  validate(evt) {
    let u = this.usernameTarget.value.trim(),
      p = this.passwordTarget.value.trim()
    if (u == "" || p == "") {
      evt.preventDefault()
      if (u == "") {
        this.usernameTarget.focus()
      } else {
        this.passwordTarget.focus()
      }
    }
  }
}
