import type { GetIdentitySettings401Code } from "./getIdentitySettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetIdentitySettings401 = Problem & {
  code?: GetIdentitySettings401Code;
};
