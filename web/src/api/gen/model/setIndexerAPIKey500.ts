import type { Problem } from "./problem.ts";
import type { SetIndexerAPIKey500Code } from "./setIndexerAPIKey500Code.ts";

export type SetIndexerAPIKey500 = Problem & {
  code?: SetIndexerAPIKey500Code;
};
