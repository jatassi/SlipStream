import type { Problem } from "./problem.ts";
import type { SetMetadataKey500Code } from "./setMetadataKey500Code.ts";

export type SetMetadataKey500 = Problem & {
  code?: SetMetadataKey500Code;
};
