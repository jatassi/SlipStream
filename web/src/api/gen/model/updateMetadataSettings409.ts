import type { Problem } from "./problem.ts";
import type { UpdateMetadataSettings409Code } from "./updateMetadataSettings409Code.ts";

export type UpdateMetadataSettings409 = Problem & {
  code?: UpdateMetadataSettings409Code;
};
