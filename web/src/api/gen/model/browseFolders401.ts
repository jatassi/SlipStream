import type { BrowseFolders401Code } from "./browseFolders401Code.ts";
import type { Problem } from "./problem.ts";

export type BrowseFolders401 = Problem & {
  code?: BrowseFolders401Code;
};
