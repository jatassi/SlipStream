import type { GetTorrent401Code } from "./getTorrent401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent401 = Problem & {
  code?: GetTorrent401Code;
};
