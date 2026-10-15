import { fileURLToPath } from "node:url";

/** The init script that marks a tab as past the startup sequence. */
export const startupPlayedScript = fileURLToPath(new URL("startup-played.js", import.meta.url));
