import type { CustomFormatSchema403Code } from "./customFormatSchema403Code.ts";
import type { Problem } from "./problem.ts";

export type CustomFormatSchema403 = Problem & {
  code?: CustomFormatSchema403Code;
};
