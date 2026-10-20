import type { Problem } from "./problem.ts";
import type { TestMetadataProvider404Code } from "./testMetadataProvider404Code.ts";

export type TestMetadataProvider404 = Problem & {
  code?: TestMetadataProvider404Code;
};
