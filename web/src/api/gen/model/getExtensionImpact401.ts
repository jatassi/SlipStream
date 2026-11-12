import type { GetExtensionImpact401Code } from "./getExtensionImpact401Code.ts";
import type { Problem } from "./problem.ts";

export type GetExtensionImpact401 = Problem & {
  code?: GetExtensionImpact401Code;
};
