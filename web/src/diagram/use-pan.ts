import { useRef } from "react";

import { DRAG_FROM } from "./drag";
import type { Client, Zoom } from "./use-view";

export type Pan = {
  /** Takes every press on the figure, a chip's included, so a press that drags a chip starts no pan. */
  readonly press: (event: PointerEvent) => void;
  /** Follows a pressed pointer that no chip drag holds; true while it pans or pinches, which previews nothing. */
  readonly follow: (event: PointerEvent) => boolean;
  /** Whether the last press moved the view, so its click selects nothing. */
  readonly moved: () => boolean;
};

type Pressing = {
  readonly points: Map<number, Client>;
  start: Client;
  moved: boolean;
};

const midpoint = ([ax, ay]: Client, [bx, by]: Client): Client => [(ax + bx) / 2, (ay + by) / 2];
const apart = ([ax, ay]: Client, [bx, by]: Client): number => Math.hypot(ax - bx, ay - by);

function pinch(zoom: Zoom, from: Client, at: Client, other: Client) {
  const [mx, my] = midpoint(from, other);
  const [nx, ny] = midpoint(at, other);
  zoom.zoomBy(apart(at, other) / apart(from, other), [nx, ny]);
  zoom.panBy(nx - mx, ny - my);
}

function move(pressing: Pressing, zoom: Zoom, event: PointerEvent, figure: HTMLElement): boolean {
  const from = pressing.points.get(event.pointerId);
  if (from === undefined) {
    return false;
  }
  const at: Client = [event.clientX, event.clientY];
  pressing.points.set(event.pointerId, at);
  const other = [...pressing.points].find(([id]) => id !== event.pointerId)?.[1];
  if (other !== undefined) {
    pressing.moved = true;
    pinch(zoom, from, at, other);
    return true;
  }
  if (!pressing.moved) {
    if (apart(pressing.start, at) <= DRAG_FROM) {
      return false;
    }
    pressing.moved = true;
    figure.setPointerCapture(event.pointerId);
    figure.dataset["panning"] = "";
  }
  zoom.panBy(at[0] - from[0], at[1] - from[1]);
  return true;
}

function lift(pressing: Pressing, event: PointerEvent, figure: HTMLElement): boolean {
  if (!pressing.points.delete(event.pointerId)) {
    return false;
  }
  if (pressing.points.size === 0) {
    delete figure.dataset["panning"];
  }
  return pressing.moved;
}

/**
 * The floor's pointers: one pressed and moved past a click pans the view,
 * and two (a pinch) zoom it about their midpoint as they spread and move it
 * as they travel.
 */
export function usePan(zoom: Zoom): Pan {
  const pressing = useRef<Pressing>({ points: new Map(), start: [0, 0], moved: false });
  return {
    press: (event) => {
      const held = pressing.current;
      const at: Client = [event.clientX, event.clientY];
      if (event.isPrimary) {
        held.points.clear();
        held.start = at;
        held.moved = false;
      }
      if (event.button === 0) {
        held.points.set(event.pointerId, at);
      }
    },
    follow: (event) => {
      const figure = event.currentTarget;
      if (!(figure instanceof HTMLElement)) {
        return false;
      }
      if (event.type === "pointermove") {
        return move(pressing.current, zoom, event, figure);
      }
      return event.type === "pointerup" || event.type === "pointercancel"
        ? lift(pressing.current, event, figure)
        : false;
    },
    moved: () => pressing.current.moved,
  };
}
