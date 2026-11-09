import type { CustomFormatSchema409Code } from "./customFormatSchema409Code.ts";
import type { Problem } from "./problem.ts";

export type CustomFormatSchema409 = Problem & {
  code?: CustomFormatSchema409Code;
};
