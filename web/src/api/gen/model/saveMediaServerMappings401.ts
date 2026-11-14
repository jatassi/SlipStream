import type { Problem } from "./problem.ts";
import type { SaveMediaServerMappings401Code } from "./saveMediaServerMappings401Code.ts";

export type SaveMediaServerMappings401 = Problem & {
  code?: SaveMediaServerMappings401Code;
};
