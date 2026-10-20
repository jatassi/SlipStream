import type { ActOnTorrents400Code } from "./actOnTorrents400Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents400 = Problem & {
  code?: ActOnTorrents400Code;
};
