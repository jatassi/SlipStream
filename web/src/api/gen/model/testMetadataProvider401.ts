import type { Problem } from "./problem.ts";
import type { TestMetadataProvider401Code } from "./testMetadataProvider401Code.ts";

export type TestMetadataProvider401 = Problem & {
  code?: TestMetadataProvider401Code;
};
