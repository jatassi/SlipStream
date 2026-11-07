import type { Problem } from "./problem.ts";
import type { SetWorksProfile401Code } from "./setWorksProfile401Code.ts";

export type SetWorksProfile401 = Problem & {
  code?: SetWorksProfile401Code;
};
