import type { Problem } from "./problem.ts";
import type { SetIndexerAPIKey403Code } from "./setIndexerAPIKey403Code.ts";

export type SetIndexerAPIKey403 = Problem & {
  code?: SetIndexerAPIKey403Code;
};
