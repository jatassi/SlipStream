import type { PasteCustomFormats400Code } from "./pasteCustomFormats400Code.ts";
import type { Problem } from "./problem.ts";

export type PasteCustomFormats400 = Problem & {
  code?: PasteCustomFormats400Code;
};
