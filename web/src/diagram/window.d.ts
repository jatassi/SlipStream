interface Window {
  /** Each recent frame's script time in ms, in development builds, for the Diagram's frame budget check. */
  __diagramTickMs?: number[];
}

/** WebKit's pinch, which a Mac trackpad sends in Safari instead of a wheel with Ctrl held. */
interface GestureEvent extends UIEvent {
  /** The spread since the gesture began, 1 at its start. */
  readonly scale: number;
  readonly clientX: number;
  readonly clientY: number;
}
