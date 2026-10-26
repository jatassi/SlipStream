/**
 * copying also covers files that are all verified and wait for their job to place them
 */
export type TransfersTransferState = "queued" | "waiting" | "needs_space" | "copying" | "failed";
