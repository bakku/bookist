import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
    static targets = ["top", "middle", "bottom"];

    connect() {
        setTimeout(() => {
            this.topTarget.style = "top: 4px; transform: rotate(45deg);";
            this.bottomTarget.style = "top: -4px; transform: rotate(-45deg);";
            this.middleTarget.style = "display: none";
        }, 1)
    }
}
