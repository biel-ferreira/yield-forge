import "@testing-library/jest-dom/vitest";

// jsdom doesn't implement <dialog>'s imperative API (SPEC-211's Dialog primitive relies on it) —
// showModal()/close()/.open are entirely missing. Polyfill the minimal behavior our component
// (components/ui/dialog.tsx) needs: toggling `.open`, calling showModal()/close(), and firing the
// native `close` event. Native semantics this can't emulate — focus trap, Escape-to-close,
// backdrop hit-testing — were proven in real-browser (Playwright) verification, not here.
if (typeof HTMLDialogElement !== "undefined" && !HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement & { open: boolean }) {
    this.open = true;
    this.setAttribute("open", "");
  };
  HTMLDialogElement.prototype.close = function (this: HTMLDialogElement & { open: boolean }) {
    if (!this.open) return;
    this.open = false;
    this.removeAttribute("open");
    this.dispatchEvent(new Event("close"));
  };
}
