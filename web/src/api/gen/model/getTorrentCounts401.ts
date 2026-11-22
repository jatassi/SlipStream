import type { GetTorrentCounts401Code } from "./getTorrentCounts401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrentCounts401 = Problem & {
  code?: GetTorrentCounts401Code;
};
