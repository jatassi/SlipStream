import type { Problem } from "./problem.ts";
import type { TestMetadataProvider422Code } from "./testMetadataProvider422Code.ts";

export type TestMetadataProvider422 = Problem & {
  code?: TestMetadataProvider422Code;
};
