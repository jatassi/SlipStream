import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume403Code } from "./updateHealthVolume403Code.ts";

export type UpdateHealthVolume403 = Problem & {
  code?: UpdateHealthVolume403Code;
};
