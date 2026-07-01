import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Merge conditional class names, de-duplicating conflicting Tailwind utilities.
 * (SPEC-200 FR-2004) — the standard `cn` helper used by every component.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
