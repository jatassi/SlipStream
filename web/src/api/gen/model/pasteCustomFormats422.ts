import type { PasteCustomFormats422Code } from "./pasteCustomFormats422Code.ts";
import type { Problem } from "./problem.ts";

export type PasteCustomFormats422 = Problem & {
  code?: PasteCustomFormats422Code;
};
