import { Controller } from "@hotwired/stimulus"

// Opens and closes the main navigation on small screens.
export default class extends Controller {
  static targets = [ "menu", "toggle" ]

  toggle() {
    const open = this.menuTarget.classList.toggle("is-open")
    this.toggleTarget.setAttribute("aria-expanded", open)
  }
}
