import type { GetIdentitySettings403Code } from "./getIdentitySettings403Code.ts";
import type { Problem } from "./problem.ts";

export type GetIdentitySettings403 = Problem & {
  code?: GetIdentitySettings403Code;
};
