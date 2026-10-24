import type { Problem } from "./problem.ts";
import type { TestMetadataProvider500Code } from "./testMetadataProvider500Code.ts";

export type TestMetadataProvider500 = Problem & {
  code?: TestMetadataProvider500Code;
};
