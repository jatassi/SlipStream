import { type RefObject, useEffect, useEffectEvent, useMemo, useRef, useState } from "react";

import type { Landing } from "../routes/_app/nodes/-landing/view";
import { lightPart, markVerdict, partAt } from "./dom";
import { type Drop, type Dragging, type GestureOf, dragFrom, dragOver, dropOf } from "./drag";
import { type Part, partsOf, stepPart } from "./parts-order";
import { useTour } from "./use-tour";

const STEPS: Readonly<Record<string, number>> = {
  ArrowRight: 1,
  ArrowDown: 1,
  ArrowLeft: -1,
  ArrowUp: -1,
};
const LISTENED = [
  "pointerdown",
  "pointermove",
  "pointerup",
  "pointercancel",
  "pointerleave",
  "click",
  "keydown",
];

export type Interaction = {
  readonly preview: Part | undefined;
  readonly dragging: Dragging | undefined;
};

type Options = {
  readonly figure: RefObject<HTMLElement | null>;
  readonly svg: RefObject<SVGSVGElement | null>;
  readonly landing: Landing;
  readonly takesDrags: boolean;
  readonly still: boolean;
  readonly selected: Part | undefined;
  readonly onSelect: (part: Part | undefined) => void;
  readonly gestureOf: GestureOf;
  readonly onDrop: (drop: Drop) => void;
};

/** The part the arrow keys, Home and End move the selection to; undefined for any other key. */
function keyedPart(key: string, parts: readonly Part[], from: Part | undefined): Part | undefined {
  if (key === "Home") {
    return parts[0];
  }
  if (key === "End") {
    return parts.at(-1);
  }
  const step = STEPS[key];
  return step === undefined ? undefined : stepPart(parts, from, step);
}

/** Calls handle with every pointer, click and key event on the element until the effect ends. */
function useListened(element: RefObject<HTMLElement | null>, handle: (event: Event) => void) {
  const onEvent = useEffectEvent(handle);
  useEffect(() => {
    const listening = new AbortController();
    for (const type of LISTENED) {
      element.current?.addEventListener(
        type,
        (event) => {
          onEvent(event);
        },
        { signal: listening.signal },
      );
    }
    return () => {
      listening.abort();
    };
  }, [element]);
}

type Drag = {
  /** The drag as last drawn, for rendering. */
  readonly dragging: Dragging | undefined;
  /** The drag as of the last event, which an event reads before React draws it. */
  readonly current: () => Dragging | undefined;
  readonly drag: (next: Dragging | undefined) => void;
  readonly release: (event: PointerEvent, held: Dragging) => void;
  /** Whether a click is the end of a drag, which selects nothing; reading it clears it. */
  readonly endsDrag: () => boolean;
};

/** A Role chip's drag: the verdict marked on the tile under it, and the drop asked for on release. */
function useDrag(svg: RefObject<SVGSVGElement | null>, onDrop: (drop: Drop) => void): Drag {
  const [dragging, setDragging] = useState<Dragging>();
  const live = useRef<Dragging>(undefined);
  const dragged = useRef(false);
  const drag = (next: Dragging | undefined) => {
    if (svg.current !== null) {
      markVerdict(svg.current, next?.to, next?.gesture?.verdict);
    }
    live.current = next;
    setDragging(next);
  };
  return {
    dragging,
    current: () => live.current,
    drag,
    release: (event, held) => {
      drag(undefined);
      dragged.current = held.moved;
      const drop = event.type === "pointerup" ? dropOf(held, event.altKey) : undefined;
      if (drop !== undefined) {
        onDrop(drop);
      }
    },
    endsDrag: () => {
      const ends = dragged.current;
      dragged.current = false;
      return ends;
    },
  };
}

type Handlers = Options & {
  readonly parts: readonly Part[];
  readonly preview: Part | undefined;
  readonly setPreview: (part: Part | undefined) => void;
  readonly hold: () => void;
  readonly drag: Drag;
};

function onPointer(
  event: PointerEvent,
  { landing, takesDrags, gestureOf, figure, preview, setPreview, hold, drag }: Handlers,
) {
  const dragging = drag.current();
  if (event.type === "pointerleave") {
    setPreview(dragging === undefined ? undefined : preview);
    return;
  }
  hold();
  const started =
    event.type === "pointerdown" && takesDrags && event.button === 0
      ? dragFrom(landing, event)
      : undefined;
  if (started !== undefined) {
    event.preventDefault();
    figure.current?.setPointerCapture(event.pointerId);
    drag.drag(started);
  } else if (dragging === undefined) {
    setPreview(event.type === "pointermove" ? partAt(event.target) : preview);
  } else if (event.type === "pointermove") {
    drag.drag(dragOver(dragging, event, gestureOf));
  } else if (event.type === "pointerup" || event.type === "pointercancel") {
    drag.release(event, dragging);
  }
}

function onKey(
  event: KeyboardEvent,
  { parts, selected, onSelect, preview, setPreview, hold, drag }: Handlers,
) {
  hold();
  if (event.key === "Escape") {
    event.preventDefault();
    drag.drag(undefined);
    setPreview(undefined);
    onSelect(undefined);
    return;
  }
  const next = keyedPart(event.key, parts, selected ?? preview);
  if (next !== undefined) {
    event.preventDefault();
    setPreview(undefined);
    onSelect(next);
  }
}

/**
 * The Diagram's pointer and keys. Hover and the idle tour preview a part;
 * a click or the arrow keys select one, and Escape clears. Where it takes
 * drags, a Role chip dragged onto another tile shows the plan's verdict and
 * a drop asks for the change.
 */
export function useInteraction(options: Options): Interaction {
  const { figure, svg, landing, selected } = options;
  const parts = useMemo(() => partsOf(landing), [landing]);
  const [preview, setPreview] = useState<Part>();
  const drag = useDrag(svg, options.onDrop);
  const hold = useTour(options.still || drag.dragging !== undefined, parts, setPreview);

  useEffect(() => {
    if (svg.current !== null && figure.current !== null) {
      lightPart(svg.current, figure.current, preview ?? selected);
    }
  });

  useListened(figure, (event) => {
    const handlers: Handlers = { ...options, parts, preview, setPreview, hold, drag };
    if (event.type === "click") {
      if (!drag.endsDrag() && event instanceof MouseEvent) {
        options.onSelect(partAt(document.elementFromPoint(event.clientX, event.clientY)));
      }
    } else if (event instanceof PointerEvent) {
      onPointer(event, handlers);
    } else if (event instanceof KeyboardEvent) {
      onKey(event, handlers);
    }
  });

  return { preview, dragging: drag.dragging };
}
