import type { GetTorrent500Code } from "./getTorrent500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent500 = Problem & {
  code?: GetTorrent500Code;
};
