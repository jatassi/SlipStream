import type { KeyboardEvent } from "react";

const step: Readonly<Record<string, (index: number, last: number) => number>> = {
  ArrowDown: (index, last) => Math.min(index + 1, last),
  ArrowUp: (index) => Math.max(index - 1, 0),
  Home: () => 0,
  End: (_, last) => last,
};

export const rowTarget = "data-row-target";
export const rowGroup = "data-row-group";

export function moveBetweenRows(event: KeyboardEvent<HTMLElement>): HTMLElement | undefined {
  const move = step[event.key];
  const group = event.currentTarget.closest(`[${rowGroup}]`);
  if (!move || !group) {
    return undefined;
  }
  const targets = [...group.querySelectorAll<HTMLElement>(`[${rowTarget}]`)];
  const index = targets.indexOf(event.currentTarget);
  event.preventDefault();
  const to = move(index, targets.length - 1);
  if (to === index) {
    return undefined;
  }
  const next = targets[to];
  next?.focus();
  return next;
}
