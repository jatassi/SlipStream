/**
 * The first that applies: off, a key is needed, the key was rejected, out of downloads or rate limited, failing, ready
 */
export type SubtitlesStandingKind = "ready" | "needs_key" | "out" | "rejected" | "failing" | "off";
