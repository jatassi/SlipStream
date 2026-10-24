import type { GetMetadataSettings500Code } from "./getMetadataSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMetadataSettings500 = Problem & {
  code?: GetMetadataSettings500Code;
};
