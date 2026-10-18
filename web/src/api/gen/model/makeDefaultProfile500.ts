import type { MakeDefaultProfile500Code } from "./makeDefaultProfile500Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultProfile500 = Problem & {
  code?: MakeDefaultProfile500Code;
};
