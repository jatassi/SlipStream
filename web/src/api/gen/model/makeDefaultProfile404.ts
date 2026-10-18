import type { MakeDefaultProfile404Code } from "./makeDefaultProfile404Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultProfile404 = Problem & {
  code?: MakeDefaultProfile404Code;
};
