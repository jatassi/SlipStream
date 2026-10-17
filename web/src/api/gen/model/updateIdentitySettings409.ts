import type { Problem } from "./problem.ts";
import type { UpdateIdentitySettings409Code } from "./updateIdentitySettings409Code.ts";

export type UpdateIdentitySettings409 = Problem & {
  code?: UpdateIdentitySettings409Code;
};
