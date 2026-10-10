import { Controller } from "@hotwired/stimulus"

// Switches between the standard palette and the all-navy one, and remembers the choice.
export default class extends Controller {
  toggle() {
    const root = document.documentElement
    root.dataset.theme = root.dataset.theme === "dark" ? "light" : "dark"
    try { localStorage.setItem("theme", root.dataset.theme) } catch {}
  }
}
