import type { Plane } from "@anatomy/iso-kit";

import type { Stage } from "./drawing";

/** The part of the stage the Diagram shows, in stage units: its viewBox. */
export type View = {
  readonly x: number;
  readonly y: number;
  readonly w: number;
  readonly h: number;
};

/** How far the Diagram magnifies its fitted view, which is the least it shows. */
export const MOST_ZOOM = 5;

/** What one press of Zoom in or Zoom out multiplies the zoom by. */
export const ZOOM_STEP = 1.5;

export const fitOf = ({ width, height }: Stage): View => ({ x: 0, y: 0, w: width, h: height });

export const zoomOf = (view: View, stage: Stage): number => stage.width / view.w;

export const isFitted = (view: View, stage: Stage): boolean => view.w >= stage.width;

export const isMost = (view: View, stage: Stage): boolean => view.w <= stage.width / MOST_ZOOM;

export const viewBoxOf = ({ x, y, w, h }: View): string =>
  [x, y, w, h].map((value) => String(Math.round(value * 100) / 100)).join(" ");

const clamp = (value: number, low: number, high: number): number =>
  Math.min(Math.max(value, low), high);

const within = (view: View, stage: Stage): View => ({
  ...view,
  x: clamp(view.x, 0, stage.width - view.w),
  y: clamp(view.y, 0, stage.height - view.h),
});

/** The view at a zoom, about a point of the stage that keeps its place on screen, kept on the stage. */
export function zoomed(view: View, stage: Stage, zoom: number, [ax, ay]: Plane): View {
  const w = stage.width / clamp(zoom, 1, MOST_ZOOM);
  const k = w / view.w;
  return within(
    {
      x: ax - (ax - view.x) * k,
      y: ay - (ay - view.y) * k,
      w,
      h: (w * stage.height) / stage.width,
    },
    stage,
  );
}

/** The view moved across the stage by a distance in stage units, kept on the stage. */
export const panned = (view: View, stage: Stage, [dx, dy]: Plane): View =>
  within({ ...view, x: view.x + dx, y: view.y + dy }, stage);

/** The view a share of the way from one view to another. */
export const between = (from: View, to: View, share: number): View => ({
  x: from.x + (to.x - from.x) * share,
  y: from.y + (to.y - from.y) * share,
  w: from.w + (to.w - from.w) * share,
  h: from.h + (to.h - from.h) * share,
});
