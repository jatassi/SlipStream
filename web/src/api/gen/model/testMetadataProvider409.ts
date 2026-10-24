import type { Problem } from "./problem.ts";
import type { TestMetadataProvider409Code } from "./testMetadataProvider409Code.ts";

export type TestMetadataProvider409 = Problem & {
  code?: TestMetadataProvider409Code;
};
