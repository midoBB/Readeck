import {Controller} from "@hotwired/stimulus"
import {request} from "../lib/request"

export default class extends Controller {
  async fetch(event) {
    if (!event.params.url) {
      throw new Error("url param is not set")
    }
    const options = {
      method: event.params.method || "get",
    }

    await request(event.params.url, options)
    this.dispatch(event.params.eventName || "done")
  }
}
