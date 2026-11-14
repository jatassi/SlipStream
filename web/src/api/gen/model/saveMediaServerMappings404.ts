import type { Problem } from "./problem.ts";
import type { SaveMediaServerMappings404Code } from "./saveMediaServerMappings404Code.ts";

export type SaveMediaServerMappings404 = Problem & {
  code?: SaveMediaServerMappings404Code;
};
