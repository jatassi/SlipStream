import type { GetTorrent409Code } from "./getTorrent409Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent409 = Problem & {
  code?: GetTorrent409Code;
};
