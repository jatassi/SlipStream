import type { ActOnTorrents401Code } from "./actOnTorrents401Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents401 = Problem & {
  code?: ActOnTorrents401Code;
};
