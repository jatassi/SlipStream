import type { GetIdentitySettings500Code } from "./getIdentitySettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetIdentitySettings500 = Problem & {
  code?: GetIdentitySettings500Code;
};
