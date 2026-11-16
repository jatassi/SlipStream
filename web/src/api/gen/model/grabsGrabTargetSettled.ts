/**
 * How the Grab finished at this Target: it placed a file, it placed none, or the Admin declined it. Null while the Grab is still downloading for it
 * @nullable
 */
export type GrabsGrabTargetSettled = "placed" | "none" | "declined" | null;
