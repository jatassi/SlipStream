import type { PasteCustomFormats500Code } from "./pasteCustomFormats500Code.ts";
import type { Problem } from "./problem.ts";

export type PasteCustomFormats500 = Problem & {
  code?: PasteCustomFormats500Code;
};
