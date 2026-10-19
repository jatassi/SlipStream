import type { BrowseFolders503Code } from "./browseFolders503Code.ts";
import type { Problem } from "./problem.ts";

export type BrowseFolders503 = Problem & {
  code?: BrowseFolders503Code;
};
