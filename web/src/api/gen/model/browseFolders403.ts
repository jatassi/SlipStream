import type { BrowseFolders403Code } from "./browseFolders403Code.ts";
import type { Problem } from "./problem.ts";

export type BrowseFolders403 = Problem & {
  code?: BrowseFolders403Code;
};
