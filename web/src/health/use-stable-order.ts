import { useState } from "react";

/**
 * The items in the order their keys were first seen while the component
 * lives, new keys after the rest in the order given, so rows never jump
 * under the pointer.
 */
export function useStableOrder<T>(items: readonly T[], keyOf: (item: T) => string): T[] {
  const byKey = new Map(items.map((item) => [keyOf(item), item]));
  const [seen, setSeen] = useState<readonly string[]>(() => [...byKey.keys()]);
  const unseen = [...byKey.keys()].filter((key) => !seen.includes(key));
  const order = unseen.length === 0 ? seen : [...seen, ...unseen];
  if (order !== seen) {
    setSeen(order);
  }
  return order.flatMap((key) => {
    const item = byKey.get(key);
    return item === undefined ? [] : [item];
  });
}
