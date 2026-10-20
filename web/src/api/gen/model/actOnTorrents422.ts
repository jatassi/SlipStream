import type { ActOnTorrents422Code } from "./actOnTorrents422Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents422 = Problem & {
  code?: ActOnTorrents422Code;
};
