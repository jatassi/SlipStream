import type { GetExtensionImpact500Code } from "./getExtensionImpact500Code.ts";
import type { Problem } from "./problem.ts";

export type GetExtensionImpact500 = Problem & {
  code?: GetExtensionImpact500Code;
};
