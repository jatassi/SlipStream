import type { GetProfile403Code } from "./getProfile403Code.ts";
import type { Problem } from "./problem.ts";

export type GetProfile403 = Problem & {
  code?: GetProfile403Code;
};
