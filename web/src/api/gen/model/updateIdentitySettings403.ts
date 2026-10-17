import type { Problem } from "./problem.ts";
import type { UpdateIdentitySettings403Code } from "./updateIdentitySettings403Code.ts";

export type UpdateIdentitySettings403 = Problem & {
  code?: UpdateIdentitySettings403Code;
};
