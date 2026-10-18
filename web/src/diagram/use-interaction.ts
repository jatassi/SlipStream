import { type RefObject, useEffect, useEffectEvent, useMemo, useRef, useState } from "react";

import type { Landing } from "../routes/_app/nodes/-landing/view";
import type { Stage } from "./drawing";
import { lightPart, markVerdict, partAt } from "./dom";
import { type Drop, type Dragging, type GestureOf, dragFrom, dragOver, dropOf } from "./drag";
import { type Part, partsOf, stepPart } from "./parts-order";
import { type Pan, usePan } from "./use-pan";
import { useTour } from "./use-tour";
import { type Zoom, useView } from "./use-view";

const STEPS: Readonly<Record<string, number>> = {
  ArrowRight: 1,
  ArrowDown: 1,
  ArrowLeft: -1,
  ArrowUp: -1,
};
const ZOOM_KEYS: Readonly<Record<string, 1 | -1 | "fit">> = {
  "+": 1,
  "=": 1,
  "-": -1,
  _: -1,
  "0": "fit",
};

// A wheel's notch is 100px and zooms by about a fifth; a trackpad pinch
// arrives as a wheel with Ctrl held, in smaller steps.
const WHEEL_RATE = 0.0018;
const PINCH_RATE = 0.01;
const LINE_PX = 16;

const LISTENED = [
  "pointerdown",
  "pointermove",
  "pointerup",
  "pointercancel",
  "pointerleave",
  "click",
  "keydown",
  "wheel",
];

export type Interaction = {
  readonly preview: Part | undefined;
  readonly dragging: Dragging | undefined;
  readonly zoom: Zoom;
};

/** What the page tells the Diagram: its Nodes, whether it takes drags, the selection and the drop. */
export type Selecting = {
  readonly landing: Landing;
  readonly takesDrags: boolean;
  readonly selected: Part | undefined;
  readonly onSelect: (part: Part | undefined) => void;
  readonly gestureOf: GestureOf;
  readonly onDrop: (drop: Drop) => void;
};

type Options = Selecting & {
  readonly figure: RefObject<HTMLElement | null>;
  readonly svg: RefObject<SVGSVGElement | null>;
  readonly stage: Stage;
  readonly still: boolean;
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
  readonly zoom: Zoom;
  readonly parts: readonly Part[];
  readonly preview: Part | undefined;
  readonly setPreview: (part: Part | undefined) => void;
  readonly hold: () => void;
  readonly drag: Drag;
  readonly pan: Pan;
};

function onPointer(
  event: PointerEvent,
  { landing, takesDrags, gestureOf, figure, preview, setPreview, hold, drag, pan }: Handlers,
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
    figure.current?.focus({ preventScroll: true });
    figure.current?.setPointerCapture(event.pointerId);
    drag.drag(started);
  } else if (dragging !== undefined) {
    if (event.type === "pointermove") {
      drag.drag(dragOver(dragging, event, gestureOf));
    } else if (event.type === "pointerup" || event.type === "pointercancel") {
      drag.release(event, dragging);
    }
  } else if (!pan.follow(event)) {
    setPreview(event.type === "pointermove" ? partAt(event.target) : preview);
  }
}

function onWheel(event: WheelEvent, { zoom, hold }: Handlers) {
  hold();
  event.preventDefault();
  const pixels = event.deltaY * (event.deltaMode === WheelEvent.DOM_DELTA_PIXEL ? 1 : LINE_PX);
  zoom.zoomBy(Math.exp(-pixels * (event.ctrlKey ? PINCH_RATE : WHEEL_RATE)), [
    event.clientX,
    event.clientY,
  ]);
}

function onKey(
  event: KeyboardEvent,
  { parts, selected, onSelect, preview, setPreview, hold, drag, zoom }: Handlers,
) {
  hold();
  const zoomKey = event.metaKey || event.ctrlKey || event.altKey ? undefined : ZOOM_KEYS[event.key];
  if (zoomKey !== undefined) {
    event.preventDefault();
    if (zoomKey === "fit") {
      zoom.fit();
    } else {
      zoom.step(zoomKey);
    }
    return;
  }
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
 * a click selects the part it pressed, the arrow keys step through them,
 * and Escape clears. Where it takes
 * drags, a Role chip dragged onto another tile shows the plan's verdict and
 * a drop asks for the change. Any other press pans the view, two pinch it,
 * and the wheel and + - 0 zoom it.
 */
export function useInteraction(options: Options): Interaction {
  const { figure, svg, landing, selected } = options;
  const parts = useMemo(() => partsOf(landing), [landing]);
  const [preview, setPreview] = useState<Part>();
  const drag = useDrag(svg, options.onDrop);
  const zoom = useView(svg, options.stage, options.still);
  const pan = usePan(zoom);
  const pressed = useRef<Part>(undefined);
  const hold = useTour(options.still || drag.dragging !== undefined, parts, setPreview);

  useEffect(() => {
    if (svg.current !== null && figure.current !== null) {
      lightPart(svg.current, figure.current, preview ?? selected);
    }
  });

  useListened(figure, (event) => {
    const handlers: Handlers = {
      ...options,
      zoom,
      parts,
      preview,
      setPreview,
      hold,
      drag,
      pan,
    };
    if (event.type === "pointerdown" && event instanceof PointerEvent) {
      pressed.current = partAt(event.target);
      pan.press(event);
    }
    if (event.type === "click") {
      if (!drag.endsDrag() && !pan.moved()) {
        options.onSelect(pressed.current);
      }
    } else if (event instanceof WheelEvent) {
      onWheel(event, handlers);
    } else if (event instanceof PointerEvent) {
      onPointer(event, handlers);
    } else if (event instanceof KeyboardEvent) {
      onKey(event, handlers);
    }
  });

  return { preview, dragging: drag.dragging, zoom };
}
