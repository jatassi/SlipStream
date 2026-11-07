import type { Problem } from "./problem.ts";
import type { SetWorksRoot500Code } from "./setWorksRoot500Code.ts";

export type SetWorksRoot500 = Problem & {
  code?: SetWorksRoot500Code;
};
