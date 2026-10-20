import type { ActOnTorrents409Code } from "./actOnTorrents409Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents409 = Problem & {
  code?: ActOnTorrents409Code;
};
