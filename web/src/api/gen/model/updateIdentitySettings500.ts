import type { Problem } from "./problem.ts";
import type { UpdateIdentitySettings500Code } from "./updateIdentitySettings500Code.ts";

export type UpdateIdentitySettings500 = Problem & {
  code?: UpdateIdentitySettings500Code;
};
