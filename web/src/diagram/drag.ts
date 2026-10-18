import type { NodesGesture } from "../api/gen/model";
import type { RoleName } from "../nodes/roles";
import type { Landing } from "../routes/_app/nodes/-landing/view";
import { chipAt, nodeAt } from "./dom";

/** A chip dropped on another tile: the Role, where it came from, where it went, and whether ⌥ added it. */
export type Drop = {
  readonly role: RoleName;
  readonly from: string;
  readonly to: string;
  readonly add: boolean;
};

/** The plan's verdict on a drop, from the role board's gestures. */
export type GestureOf = (
  role: RoleName,
  from: string,
  to: string,
  add: boolean,
) => NodesGesture | undefined;

export type Dragging = {
  readonly role: RoleName;
  readonly from: string;
  readonly label: string;
  readonly start: readonly [number, number];
  readonly at: readonly [number, number];
  /** Past the few pixels that tell a drag from a click. */
  readonly moved: boolean;
  readonly add: boolean;
  readonly to?: string;
  readonly gesture?: NodesGesture;
};

/** The pixels a press moves before it drags rather than clicks. */
export const DRAG_FROM = 4;

type Pointer = Pick<PointerEvent, "clientX" | "clientY" | "altKey" | "target">;

/** The drag a pointer going down on a Role chip starts. */
export function dragFrom(landing: Landing, event: Pointer): Dragging | undefined {
  const chip = chipAt(
    event.target,
    (name) => landing.nodes.flatMap(({ roles }) => roles).find(({ role }) => role === name)?.role,
  );
  if (chip === undefined) {
    return undefined;
  }
  const label =
    landing.nodes.find(({ id }) => id === chip.from)?.roles.find(({ role }) => role === chip.role)
      ?.label ?? chip.role;
  const at = [event.clientX, event.clientY] as const;
  return { ...chip, label, start: at, at, moved: false, add: event.altKey };
}

/** The drag as the pointer moves: the tile under it, and the plan's verdict on dropping there. */
export function dragOver(dragging: Dragging, event: Pointer, gestureOf: GestureOf): Dragging {
  const at = [event.clientX, event.clientY] as const;
  const moved =
    dragging.moved || Math.hypot(at[0] - dragging.start[0], at[1] - dragging.start[1]) > DRAG_FROM;
  const over = moved ? nodeAt(at[0], at[1]) : undefined;
  const to = over === dragging.from ? undefined : over;
  const gesture =
    to === undefined ? undefined : gestureOf(dragging.role, dragging.from, to, event.altKey);
  const { role, from, label, start } = dragging;
  return {
    role,
    from,
    label,
    start,
    at,
    moved,
    add: event.altKey,
    ...(to === undefined ? {} : { to }),
    ...(gesture === undefined ? {} : { gesture }),
  };
}

/** The drop a released drag makes: none on its own tile, off every tile, or where the plan blocks it. */
export function dropOf(
  { role, from, to, gesture, moved }: Dragging,
  add: boolean,
): Drop | undefined {
  return moved && to !== undefined && gesture !== undefined && gesture.verdict !== "block"
    ? { role, from, to, add }
    : undefined;
}
