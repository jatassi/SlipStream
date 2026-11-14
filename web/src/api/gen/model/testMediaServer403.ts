import type { Problem } from "./problem.ts";
import type { TestMediaServer403Code } from "./testMediaServer403Code.ts";

export type TestMediaServer403 = Problem & {
  code?: TestMediaServer403Code;
};
