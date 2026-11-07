import type { Problem } from "./problem.ts";
import type { SetWorksRoot401Code } from "./setWorksRoot401Code.ts";

export type SetWorksRoot401 = Problem & {
  code?: SetWorksRoot401Code;
};
