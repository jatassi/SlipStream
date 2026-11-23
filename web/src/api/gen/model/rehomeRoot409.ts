import type { Problem } from "./problem.ts";
import type { RehomeRoot409Code } from "./rehomeRoot409Code.ts";
import type { RootsBusy } from "./rootsBusy.ts";
import type { RootsMissingFiles } from "./rootsMissingFiles.ts";

export type RehomeRoot409 = Problem & {
  busy?: RootsBusy;
  code?: RehomeRoot409Code;
  missing?: RootsMissingFiles;
};
