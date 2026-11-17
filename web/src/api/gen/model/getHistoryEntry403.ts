import type { GetHistoryEntry403Code } from "./getHistoryEntry403Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry403 = Problem & {
  code?: GetHistoryEntry403Code;
};
