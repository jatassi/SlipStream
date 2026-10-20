import type { GetTorrent422Code } from "./getTorrent422Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent422 = Problem & {
  code?: GetTorrent422Code;
};
