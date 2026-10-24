import type { Problem } from "./problem.ts";
import type { SetMetadataKey404Code } from "./setMetadataKey404Code.ts";

export type SetMetadataKey404 = Problem & {
  code?: SetMetadataKey404Code;
};
