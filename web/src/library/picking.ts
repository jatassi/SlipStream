import {
  type KeyboardEvent,
  type MouseEvent,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { noPicks, toggle, toggleAll } from "../ui/picks";
import { type Step, stepOf } from "./keyboard";
import type { LibraryWork } from "./rows";

/** A function whose identity holds across renders and that always runs the latest one. */
function useStable<A extends unknown[], R>(fn: (...args: A) => R): (...args: A) => R {
  const latest = useRef(fn);
  useLayoutEffect(() => {
    latest.current = fn;
  });
  return useCallback((...args: A) => latest.current(...args), []);
}

/** The Works picked for bulk edit, which survive filter changes. */
export type Picking = {
  /** Clicks pick rather than open, from the first pick or once Select is chosen. */
  readonly selecting: boolean;
  readonly has: (id: string) => boolean;
  /** Every picked Work still in this Library, shown or not. */
  readonly picked: readonly LibraryWork[];
  /** The picked Works the filters or the search hide. */
  readonly hidden: number;
  readonly toggle: (id: string, range: boolean) => void;
  /** Picks every Work the filters show, or none when all of them already are. */
  readonly toggleAll: () => void;
  readonly select: () => void;
  readonly done: () => void;
};

export function usePicking(
  works: ReadonlyMap<string, LibraryWork>,
  order: readonly string[],
): Picking {
  const [picks, setPicks] = useState(noPicks);
  const [chosen, setChosen] = useState(false);
  const shownOrder = useRef(order);
  useLayoutEffect(() => {
    shownOrder.current = order;
  });
  const picked = [...picks.ids].flatMap((id) => {
    const work = works.get(id);
    return work === undefined ? [] : [work];
  });
  const shown = new Set(order);
  const togglePick = useCallback((id: string, range: boolean) => {
    setPicks((current) => toggle(current, id, shownOrder.current, range));
  }, []);
  return {
    selecting: chosen || picked.length > 0,
    has: (id) => picks.ids.has(id),
    picked,
    hidden: picked.filter(({ id }) => !shown.has(id)).length,
    toggle: togglePick,
    toggleAll: () => {
      setChosen(true);
      setPicks((current) => toggleAll(current, shownOrder.current));
    },
    select: () => {
      setChosen(true);
    },
    done: () => {
      setChosen(false);
      setPicks(noPicks);
    },
  };
}

/** What a grid or table cell calls, the same function on every render. */
export type CellHandlers = {
  readonly onClick: (event: MouseEvent, id: string) => void;
  readonly onKeyDown: (event: KeyboardEvent, id: string) => void;
  readonly onFocus: (id: string) => void;
};

export const focusTarget = "data-focus-target";

const targetOf = (id: string) =>
  document.querySelector<HTMLElement>(`[data-work-id="${CSS.escape(id)}"] [${focusTarget}]`);

// Focuses a Work once it is drawn, which may be a render or two after it scrolls into view.
function usePendingFocus() {
  const pending = useRef<string>(undefined);
  useEffect(() => {
    const id = pending.current;
    const target = id === undefined ? null : targetOf(id);
    if (target !== null) {
      pending.current = undefined;
      target.focus();
    }
  });
  return (id: string) => {
    pending.current = id;
  };
}

/**
 * One Work at a time takes Tab, and the arrow keys move it, scrolling the
 * Work into view first when it isn't drawn yet. Enter opens a Work, Space
 * picks it, and while picking a click picks too, with Shift for a range.
 */
export function useBrowsing({
  order,
  picking,
  move,
  reveal,
}: {
  order: readonly string[];
  picking: Picking;
  move: (index: number, step: Step, count: number) => number;
  reveal: (index: number) => void;
}): { tabbable: string | undefined; handlers: CellHandlers } {
  const [focused, setFocused] = useState<string>();
  const focusSoon = usePendingFocus();
  const onKeyDown = useStable((event: KeyboardEvent, id: string) => {
    if (event.key === " ") {
      event.preventDefault();
      picking.toggle(id, event.shiftKey);
      return;
    }
    const step = stepOf(event.key);
    if (step === undefined) {
      return;
    }
    event.preventDefault();
    const to = move(order.indexOf(id), step, order.length);
    const next = order[to];
    if (next !== undefined && next !== id) {
      focusSoon(next);
      setFocused(next);
      reveal(to);
    }
  });
  const onClick = useStable((event: MouseEvent, id: string) => {
    if (picking.selecting || event.shiftKey) {
      event.preventDefault();
      picking.toggle(id, event.shiftKey);
    }
  });
  const onFocus = useStable((id: string) => {
    setFocused(id);
  });
  const tabbable = focused !== undefined && order.includes(focused) ? focused : order[0];
  return { tabbable, handlers: { onClick, onKeyDown, onFocus } };
}
