import type { GetNodeConfig403Code } from "./getNodeConfig403Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConfig403 = Problem & {
  code?: GetNodeConfig403Code;
};
