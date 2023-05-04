import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
    static targets = ["top", "middle", "bottom"];

    static values = {
        animationType: String,
    }

    connect() {
        setTimeout(() => {
            if (this.animationTypeValue === "open") this.openAnimation();
            else this.closeAnimation();
        }, 1)
    }

    openAnimation() {
        this.topTarget.style = "top: 4px; transform: rotate(45deg);";
        this.bottomTarget.style = "top: -4px; transform: rotate(-45deg);";
        this.middleTarget.style = "display: none";
    }

    closeAnimation() {
        this.topTarget.style = "top: 0px; transform: rotate(0deg);";
        this.bottomTarget.style = "top: 0px; transform: rotate(0deg);";

        // Tailwind lets a transition take 150ms by default, so halfway
        // through we are going to show the middle line again
        setTimeout(() => {
            this.middleTarget.style = "display: block";
        }, 75)
    }
}
