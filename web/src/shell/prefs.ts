import { useSyncExternalStore } from "react";

type Area = "browser" | "tab";

type Values = {
  sidebarCollapsed: boolean;
  startupOff: boolean;
  startupPlayed: boolean;
  lastEmail: string;
  showHiddenFolders: boolean;
  setupCode: string;
};

export type Pref = keyof Values;

type Kind<T> = { area: Area; fallback: T; parse: (raw: string) => T };

const flag = (area: Area): Kind<boolean> => ({
  area,
  fallback: false,
  parse: (raw) => raw === "true",
});

const text = (area: Area): Kind<string> => ({ area, fallback: "", parse: (raw) => raw });

const kinds: { readonly [P in Pref]: Kind<Values[P]> } = {
  sidebarCollapsed: flag("browser"),
  startupOff: flag("browser"),
  startupPlayed: flag("tab"),
  lastEmail: text("browser"),
  showHiddenFolders: flag("browser"),
  setupCode: text("tab"),
};

export const prefKey = (pref: Pref): string => `slipstream.${pref}`;

const storageOf = (pref: Pref): Storage =>
  kinds[pref].area === "browser" ? localStorage : sessionStorage;

const values: { [P in Pref]?: Values[P] } = {};
const listeners = new Map<Pref, Set<() => void>>();

function stored<P extends Pref>(pref: P): Values[P] {
  const kind: Kind<Values[P]> = kinds[pref];
  try {
    const raw = storageOf(pref).getItem(prefKey(pref));
    return raw === null ? kind.fallback : kind.parse(raw);
  } catch {
    return kind.fallback;
  }
}

// Private windows can refuse storage, so a failed read gives the fallback and
// a failed write holds for this page only.
export function readPref<P extends Pref>(pref: P): Values[P] {
  let value: Values[P] | undefined = values[pref];
  if (value === undefined) {
    value = stored(pref);
    values[pref] = value;
  }
  return value;
}

export function writePref<P extends Pref>(pref: P, value: Values[P]): void {
  values[pref] = value;
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

export function usePref<P extends Pref>(pref: P): Values[P] {
  return useSyncExternalStore(
    (listener) => subscribePref(pref, listener),
    () => readPref(pref),
  );
}
