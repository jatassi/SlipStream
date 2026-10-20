import type { GetTorrent404Code } from "./getTorrent404Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent404 = Problem & {
  code?: GetTorrent404Code;
};
