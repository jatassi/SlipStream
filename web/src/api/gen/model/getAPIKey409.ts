import type { GetAPIKey409Code } from "./getAPIKey409Code.ts";
import type { Problem } from "./problem.ts";

export type GetAPIKey409 = Problem & {
  code?: GetAPIKey409Code;
};
