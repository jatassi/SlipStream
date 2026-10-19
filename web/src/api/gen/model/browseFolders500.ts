import type { BrowseFolders500Code } from "./browseFolders500Code.ts";
import type { Problem } from "./problem.ts";

export type BrowseFolders500 = Problem & {
  code?: BrowseFolders500Code;
};
