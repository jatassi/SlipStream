import { useSyncExternalStore } from "react";

import type { ShellKind } from "../ui/shell-context";

export const desktopQuery = "(min-width: 768px)";
export const narrowDesktopQuery = "(max-width: 1099px)";

export function useMedia(query: string): boolean {
  return useSyncExternalStore(
    (listener) => {
      const list = matchMedia(query);
      list.addEventListener("change", listener);
      return () => {
        list.removeEventListener("change", listener);
      };
    },
    () => matchMedia(query).matches,
  );
}

export const useShellKind = (): ShellKind => (useMedia(desktopQuery) ? "desktop" : "phone");

export const densityOf = { desktop: "console", phone: "touch" } as const satisfies Record<
  ShellKind,
  string
>;

/** Wide enough for the Nodes Diagram to be the default view and to take drags. */
export const diagramQuery = "(min-width: 1000px)";
