import type { GetAPIKey403Code } from "./getAPIKey403Code.ts";
import type { Problem } from "./problem.ts";

export type GetAPIKey403 = Problem & {
  code?: GetAPIKey403Code;
};
