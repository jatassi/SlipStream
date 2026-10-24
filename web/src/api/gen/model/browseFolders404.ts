import type { BrowseFolders404Code } from "./browseFolders404Code.ts";
import type { Problem } from "./problem.ts";

export type BrowseFolders404 = Problem & {
  code?: BrowseFolders404Code;
};
