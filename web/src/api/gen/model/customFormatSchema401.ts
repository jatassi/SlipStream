import type { CustomFormatSchema401Code } from "./customFormatSchema401Code.ts";
import type { Problem } from "./problem.ts";

export type CustomFormatSchema401 = Problem & {
  code?: CustomFormatSchema401Code;
};
