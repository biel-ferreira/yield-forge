"use client";

import { useEffect, useSyncExternalStore } from "react";

// Dark-first; the choice is persisted. Light drops the aurora glow (see globals.css).
// The current theme is read from the DOM via useSyncExternalStore (idiomatic external
// store), so there is no setState-in-effect cascading render.

function subscribe(callback: () => void) {
  window.addEventListener("yf-theme-change", callback);
  return () => window.removeEventListener("yf-theme-change", callback);
}
const getSnapshot = () =>
  document.documentElement.getAttribute("data-theme") === "light" ? "light" : "dark";
const getServerSnapshot = () => "dark" as const;

export function ThemeToggle() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const light = theme === "light";

  // Restore the persisted theme on mount — DOM-only (no setState → no cascading render).
  useEffect(() => {
    try {
      if (localStorage.getItem("yf-theme") === "light") {
        document.documentElement.setAttribute("data-theme", "light");
        window.dispatchEvent(new Event("yf-theme-change"));
      }
    } catch {
      // localStorage unavailable — stay on the default dark theme
    }
  }, []);

  function toggle() {
    const next = light ? "dark" : "light";
    const el = document.documentElement;
    if (next === "light") el.setAttribute("data-theme", "light");
    else el.removeAttribute("data-theme");
    try {
      localStorage.setItem("yf-theme", next);
    } catch {
      // ignore persistence failure
    }
    window.dispatchEvent(new Event("yf-theme-change"));
  }

  return (
    <button
      onClick={toggle}
      aria-label="Alternar tema"
      className="rounded-full border border-hairline bg-surface px-3 py-1.5 text-xs font-semibold text-muted-strong transition-colors hover:text-on-dark"
    >
      {light ? "☀ Claro" : "☾ Escuro"}
    </button>
  );
}
