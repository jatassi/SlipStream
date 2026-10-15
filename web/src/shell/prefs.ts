import { useSyncExternalStore } from "react";

type Area = "browser" | "tab";

const areas = {
  sidebarCollapsed: "browser",
  startupOff: "browser",
  startupPlayed: "tab",
} as const satisfies Record<string, Area>;

export type Pref = keyof typeof areas;

export const prefKey = (pref: Pref): string => `slipstream.${pref}`;

const storageOf = (pref: Pref): Storage =>
  areas[pref] === "browser" ? localStorage : sessionStorage;

const values = new Map<Pref, boolean>();
const listeners = new Map<Pref, Set<() => void>>();

// Private windows can refuse storage, so a failed read gives false and a
// failed write holds for this page only.
export function readPref(pref: Pref): boolean {
  let value = values.get(pref);
  if (value === undefined) {
    try {
      value = storageOf(pref).getItem(prefKey(pref)) === "true";
    } catch {
      value = false;
    }
    values.set(pref, value);
  }
  return value;
}

export function writePref(pref: Pref, value: boolean): void {
  values.set(pref, value);
  try {
    storageOf(pref).setItem(prefKey(pref), String(value));
  } catch {
    // The value holds for this page.
  }
  for (const listener of listeners.get(pref) ?? []) {
    listener();
  }
}

export function subscribePref(pref: Pref, listener: () => void): () => void {
  const set = listeners.get(pref) ?? new Set();
  listeners.set(pref, set);
  set.add(listener);
  return () => {
    set.delete(listener);
  };
}

export function usePref(pref: Pref): boolean {
  return useSyncExternalStore(
    (listener) => subscribePref(pref, listener),
    () => readPref(pref),
  );
}
