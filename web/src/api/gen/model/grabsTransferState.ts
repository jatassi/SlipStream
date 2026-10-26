/**
 * copying also covers a file that is verified and waits to be imported
 */
export type GrabsTransferState = "queued" | "waiting" | "needs_space" | "copying" | "failed";
