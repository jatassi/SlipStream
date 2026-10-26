import type { Problem } from "./problem.ts";
import type { SetWorksMonitored500Code } from "./setWorksMonitored500Code.ts";

export type SetWorksMonitored500 = Problem & {
  code?: SetWorksMonitored500Code;
};
