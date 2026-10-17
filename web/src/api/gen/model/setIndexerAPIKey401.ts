import type { Problem } from "./problem.ts";
import type { SetIndexerAPIKey401Code } from "./setIndexerAPIKey401Code.ts";

export type SetIndexerAPIKey401 = Problem & {
  code?: SetIndexerAPIKey401Code;
};
