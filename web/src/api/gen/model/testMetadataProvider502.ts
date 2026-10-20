import type { Problem } from "./problem.ts";
import type { TestMetadataProvider502Code } from "./testMetadataProvider502Code.ts";

export type TestMetadataProvider502 = Problem & {
  code?: TestMetadataProvider502Code;
};
