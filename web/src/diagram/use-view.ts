import type { Plane } from "@anatomy/iso-kit";
import { type RefObject, useEffect, useRef, useState } from "react";

import type { Stage } from "./drawing";
import {
  type View,
  ZOOM_STEP,
  between,
  fitOf,
  isFitted,
  isMost,
  panned,
  zoomOf,
  zoomed,
} from "./view";

const EASE_MS = 220;

/** A point on the screen, in client pixels. */
export type Client = readonly [number, number];

export type Zoom = {
  readonly view: View;
  readonly fitted: boolean;
  readonly most: boolean;
  /** Multiplies the zoom about a point on screen, at once. */
  readonly zoomBy: (factor: number, at: Client) => void;
  /** Moves the drawing with the pointer, by a distance in client pixels. */
  readonly panBy: (dx: number, dy: number) => void;
  /** One step in or out about the middle of the view, eased unless motion is reduced. */
  readonly step: (direction: 1 | -1) => void;
  /** Back to the fitted view, eased unless motion is reduced. */
  readonly fit: () => void;
};

type Held = { readonly stage: Stage; readonly view: View };

const sameSize = (a: Stage, b: Stage): boolean => a.width === b.width && a.height === b.height;

// A new drawing of the same size keeps the view (a Role moved), and one of
// another size (a Node added or gone) fits again.
const viewOf = (held: Held, stage: Stage): View =>
  sameSize(held.stage, stage) ? held.view : fitOf(stage);

function boxOf(svg: RefObject<SVGSVGElement | null>): DOMRect {
  if (svg.current === null) {
    throw new Error("the Diagram zoomed before its drawing mounted");
  }
  return svg.current.getBoundingClientRect();
}

/** Where a point on screen falls on the stage. */
const stageAt = (box: DOMRect, view: View, [cx, cy]: Client): Plane => [
  view.x + ((cx - box.left) * view.w) / box.width,
  view.y + ((cy - box.top) * view.w) / box.width,
];

/** Sets the view, easing to it over EASE_MS unless motion is reduced; a later change stops an ease. */
function useEase(still: boolean, show: (view: View) => void) {
  const easing = useRef(0);
  useEffect(
    () => () => {
      cancelAnimationFrame(easing.current);
    },
    [],
  );
  const stop = () => {
    cancelAnimationFrame(easing.current);
  };
  const easeTo = (from: View, to: View) => {
    stop();
    if (still) {
      show(to);
      return;
    }
    let started: number | undefined;
    const frame = (now: number) => {
      started ??= now;
      const t = Math.min(1, (now - started) / EASE_MS);
      show(between(from, to, 1 - (1 - t) ** 3));
      if (t < 1) {
        easing.current = requestAnimationFrame(frame);
      }
    };
    easing.current = requestAnimationFrame(frame);
  };
  return { stop, easeTo };
}

/**
 * The Diagram's zoom and pan, as the svg's viewBox over the drawn stage: the
 * fitted view at the least, up to MOST_ZOOM, never past the stage's edges.
 */
export function useView(svg: RefObject<SVGSVGElement | null>, stage: Stage, still: boolean): Zoom {
  const [held, setHeld] = useState<Held>(() => ({ stage, view: fitOf(stage) }));
  const { stop, easeTo } = useEase(still, (view) => {
    setHeld({ stage, view });
  });
  const view = viewOf(held, stage);
  const change = (next: (prior: View) => View) => {
    stop();
    setHeld((prior) => ({ stage, view: next(viewOf(prior, stage)) }));
  };
  return {
    view,
    fitted: isFitted(view, stage),
    most: isMost(view, stage),
    zoomBy: (factor, client) => {
      const box = boxOf(svg);
      change((prior) =>
        zoomed(prior, stage, zoomOf(prior, stage) * factor, stageAt(box, prior, client)),
      );
    },
    panBy: (dx, dy) => {
      const box = boxOf(svg);
      change((prior) => {
        const unit = prior.w / box.width;
        return panned(prior, stage, [-dx * unit, -dy * unit]);
      });
    },
    step: (direction) => {
      const middle: Plane = [view.x + view.w / 2, view.y + view.h / 2];
      easeTo(view, zoomed(view, stage, zoomOf(view, stage) * ZOOM_STEP ** direction, middle));
    },
    fit: () => {
      easeTo(view, fitOf(stage));
    },
  };
}
