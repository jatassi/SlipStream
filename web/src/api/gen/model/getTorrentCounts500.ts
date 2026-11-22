import type { GetTorrentCounts500Code } from "./getTorrentCounts500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrentCounts500 = Problem & {
  code?: GetTorrentCounts500Code;
};
