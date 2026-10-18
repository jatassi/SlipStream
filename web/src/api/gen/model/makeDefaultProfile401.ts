import type { MakeDefaultProfile401Code } from "./makeDefaultProfile401Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultProfile401 = Problem & {
  code?: MakeDefaultProfile401Code;
};
