import type { Problem } from "./problem.ts";
import type { RestartNode403Code } from "./restartNode403Code.ts";

export type RestartNode403 = Problem & {
  code?: RestartNode403Code;
};
