import type { Problem } from "./problem.ts";
import type { SetMetadataKey422Code } from "./setMetadataKey422Code.ts";

export type SetMetadataKey422 = Problem & {
  code?: SetMetadataKey422Code;
};
