import type { Problem } from "./problem.ts";
import type { UpdateMetadataSettings401Code } from "./updateMetadataSettings401Code.ts";

export type UpdateMetadataSettings401 = Problem & {
  code?: UpdateMetadataSettings401Code;
};
