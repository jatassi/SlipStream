import type { Problem } from "./problem.ts";
import type { SaveMediaServerMappings500Code } from "./saveMediaServerMappings500Code.ts";

export type SaveMediaServerMappings500 = Problem & {
  code?: SaveMediaServerMappings500Code;
};
