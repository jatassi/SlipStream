import type { Problem } from "./problem.ts";
import type { UpdateIdentitySettings401Code } from "./updateIdentitySettings401Code.ts";

export type UpdateIdentitySettings401 = Problem & {
  code?: UpdateIdentitySettings401Code;
};
