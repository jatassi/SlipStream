import type { Problem } from "./problem.ts";
import type { SetIndexerAPIKey404Code } from "./setIndexerAPIKey404Code.ts";

export type SetIndexerAPIKey404 = Problem & {
  code?: SetIndexerAPIKey404Code;
};
