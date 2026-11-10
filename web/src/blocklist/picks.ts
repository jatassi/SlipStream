import { useState } from "react";

import { noPicks, toggle, toggleAll } from "../ui/picks";

export function usePicks(order: readonly string[]) {
  const [picks, setPicks] = useState(noPicks);
  const picked = order.filter((hash) => picks.ids.has(hash));
  return {
    picked,
    picking: {
      has: (hash: string) => picks.ids.has(hash),
      toggle: (hash: string, range: boolean) => {
        setPicks(toggle(picks, hash, order, range));
      },
    },
    all: picked.length === 0 ? "none" : picked.length === order.length ? "every" : "some",
    toggleAll: () => {
      setPicks(toggleAll(picks, order));
    },
    clear: () => {
      setPicks(noPicks);
    },
  } as const;
}
