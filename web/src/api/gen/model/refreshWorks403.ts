import type { Problem } from "./problem.ts";
import type { RefreshWorks403Code } from "./refreshWorks403Code.ts";

export type RefreshWorks403 = Problem & {
  code?: RefreshWorks403Code;
};
