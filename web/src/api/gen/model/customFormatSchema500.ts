import type { CustomFormatSchema500Code } from "./customFormatSchema500Code.ts";
import type { Problem } from "./problem.ts";

export type CustomFormatSchema500 = Problem & {
  code?: CustomFormatSchema500Code;
};
