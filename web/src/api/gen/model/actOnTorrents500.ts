import type { ActOnTorrents500Code } from "./actOnTorrents500Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents500 = Problem & {
  code?: ActOnTorrents500Code;
};
