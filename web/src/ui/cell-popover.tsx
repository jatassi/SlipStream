import { Popover as BasePopover } from "@base-ui/react/popover";
import { type ReactNode, useCallback, useEffect, useState } from "react";

import { detailPopup } from "./popover";
import { usePortalDensity } from "./portal-density";

const openDelay = 120;
const closeDelay = 160;

type Anchored<C> = { readonly el: HTMLElement; readonly cell: C; readonly pinned: boolean };

/** Closed keeps the last cell, so the popup can fade out with its content. */
type Shown<C> = { readonly open: boolean; readonly at: Anchored<C> | null };

export type CellPopoverControls<C> = {
  /** Hover or focus on a cell: open after a beat, or move at once if open. */
  peek: (el: HTMLElement, cell: C) => void;
  /** The pointer or focus left the cell or the popup. */
  leave: () => void;
  /** The pointer came back into the popup. */
  stay: () => void;
  /** A click pins the cell's popover, and a second click on it unpins. */
  pin: (el: HTMLElement, cell: C) => void;
  close: () => void;
};

export type CellPopoverState<C> = {
  readonly shown: Shown<C>;
  /** The same object for the page's life, so a memoized row never redraws for it. */
  readonly controls: CellPopoverControls<C>;
};

function controlsFor<C>(publish: (shown: Shown<C>) => void) {
  let shown: Shown<C> = { open: false, at: null };
  let timer: ReturnType<typeof setTimeout> | undefined;
  const cancel = () => {
    clearTimeout(timer);
  };
  const set = (next: Shown<C>) => {
    cancel();
    shown = next;
    publish(next);
  };
  const later = (ms: number, next: Shown<C>) => {
    cancel();
    timer = setTimeout(() => {
      set(next);
    }, ms);
  };
  const pinned = () => shown.open && shown.at?.pinned === true;
  const hidden = () => ({ ...shown, open: false });
  const controls: CellPopoverControls<C> = {
    peek(el, cell) {
      const next = { open: true, at: { el, cell, pinned: false } };
      if (pinned()) {
        return;
      }
      if (shown.open) {
        set(next);
      } else {
        later(openDelay, next);
      }
    },
    leave() {
      if (!pinned()) {
        later(closeDelay, hidden());
      }
    },
    stay: cancel,
    pin(el, cell) {
      set(
        pinned() && shown.at?.el === el ? hidden() : { open: true, at: { el, cell, pinned: true } },
      );
    },
    close() {
      set(hidden());
    },
  };
  return { controls, dispose: cancel };
}

/** One popover for every cell on a page. Rows spread cellTrigger on each cell. */
export function useCellPopover<C>(): CellPopoverState<C> {
  const [shown, setShown] = useState<Shown<C>>({ open: false, at: null });
  const [{ controls, dispose }] = useState(() => controlsFor<C>(setShown));
  useEffect(() => dispose, [dispose]);
  return { shown, controls };
}

export function cellTrigger<C>(controls: CellPopoverControls<C>, cell: C) {
  const peek = ({ currentTarget }: { currentTarget: HTMLElement }) => {
    controls.peek(currentTarget, cell);
  };
  return {
    onPointerEnter: peek,
    onFocus: peek,
    onPointerLeave: controls.leave,
    onBlur: controls.leave,
    onClick: ({ currentTarget }: { currentTarget: HTMLElement }) => {
      controls.pin(currentTarget, cell);
    },
  };
}

/**
 * A press on the anchored cell, or focus moving back to it, comes before that
 * cell's click, which unpins. Closing here would make the click pin again.
 */
function towardsAnchor(details: BasePopover.Root.ChangeEventDetails, anchor: HTMLElement) {
  if (details.reason !== "outside-press" && details.reason !== "focus-out") {
    return false;
  }
  const { event } = details;
  const to = event instanceof FocusEvent ? event.relatedTarget : event.target;
  return to instanceof Node && anchor.contains(to);
}

function dismissalOf<C>({ shown: { open, at }, controls }: CellPopoverState<C>) {
  return (next: boolean, details: BasePopover.Root.ChangeEventDetails) => {
    if (next || at === null || towardsAnchor(details, at.el)) {
      return;
    }
    if (details.reason === "escape-key" && open && at.pinned) {
      at.el.focus();
    }
    controls.close();
  };
}

export function CellPopover<C>({
  state,
  label,
  children,
}: {
  state: CellPopoverState<C>;
  label: (cell: C) => string;
  children: (cell: C) => ReactNode;
}) {
  const { density, probe } = usePortalDensity();
  const { open, at } = state.shown;
  const pinned = open && at?.pinned === true;
  const focusWhenPinned = useCallback(
    (popup: HTMLDivElement | null) => {
      if (pinned) {
        popup?.focus();
      }
    },
    [pinned],
  );
  return (
    <>
      {probe}
      <BasePopover.Root open={open} onOpenChange={dismissalOf(state)}>
        <BasePopover.Portal data-density={density}>
          <BasePopover.Positioner anchor={at?.el} sideOffset={6} collisionPadding={12}>
            <BasePopover.Popup
              ref={focusWhenPinned}
              aria-label={at === null ? undefined : label(at.cell)}
              initialFocus={false}
              finalFocus={false}
              onPointerEnter={state.controls.stay}
              onPointerLeave={state.controls.leave}
              className={detailPopup}
            >
              {at === null ? null : children(at.cell)}
            </BasePopover.Popup>
          </BasePopover.Positioner>
        </BasePopover.Portal>
      </BasePopover.Root>
    </>
  );
}
