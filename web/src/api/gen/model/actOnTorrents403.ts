import type { ActOnTorrents403Code } from "./actOnTorrents403Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnTorrents403 = Problem & {
  code?: ActOnTorrents403Code;
};
