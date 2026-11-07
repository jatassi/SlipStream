import type { Problem } from "./problem.ts";
import type { SetWorksRoot404Code } from "./setWorksRoot404Code.ts";

export type SetWorksRoot404 = Problem & {
  code?: SetWorksRoot404Code;
};
