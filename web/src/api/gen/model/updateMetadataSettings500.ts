import type { Problem } from "./problem.ts";
import type { UpdateMetadataSettings500Code } from "./updateMetadataSettings500Code.ts";

export type UpdateMetadataSettings500 = Problem & {
  code?: UpdateMetadataSettings500Code;
};
