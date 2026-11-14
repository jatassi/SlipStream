/**
 * How the Grab finished with the cell: a file placed into it, no file filled it, or the Admin declined it. Null while the Grab holds it
 * @nullable
 */
export type GrabsGrabTargetSettled = "placed" | "none" | "declined" | null;
