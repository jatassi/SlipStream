import type { Problem } from "./problem.ts";
import type { SetWorksMonitored401Code } from "./setWorksMonitored401Code.ts";

export type SetWorksMonitored401 = Problem & {
  code?: SetWorksMonitored401Code;
};
