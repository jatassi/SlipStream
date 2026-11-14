import type { Problem } from "./problem.ts";
import type { SaveMediaServerMappings403Code } from "./saveMediaServerMappings403Code.ts";

export type SaveMediaServerMappings403 = Problem & {
  code?: SaveMediaServerMappings403Code;
};
