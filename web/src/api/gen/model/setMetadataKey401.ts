import type { Problem } from "./problem.ts";
import type { SetMetadataKey401Code } from "./setMetadataKey401Code.ts";

export type SetMetadataKey401 = Problem & {
  code?: SetMetadataKey401Code;
};
