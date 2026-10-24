import type { Problem } from "./problem.ts";
import type { TestMetadataProvider403Code } from "./testMetadataProvider403Code.ts";

export type TestMetadataProvider403 = Problem & {
  code?: TestMetadataProvider403Code;
};
