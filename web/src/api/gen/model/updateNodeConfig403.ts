import type { Problem } from "./problem.ts";
import type { UpdateNodeConfig403Code } from "./updateNodeConfig403Code.ts";

export type UpdateNodeConfig403 = Problem & {
  code?: UpdateNodeConfig403Code;
};
