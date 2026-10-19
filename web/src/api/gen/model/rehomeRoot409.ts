import type { Problem } from "./problem.ts";
import type { RehomeRoot409Code } from "./rehomeRoot409Code.ts";
import type { RootsMissingFiles } from "./rootsMissingFiles.ts";

export type RehomeRoot409 = Problem & {
  code?: RehomeRoot409Code;
  missing?: RootsMissingFiles;
};
