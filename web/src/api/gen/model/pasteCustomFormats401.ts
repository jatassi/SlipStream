import type { PasteCustomFormats401Code } from "./pasteCustomFormats401Code.ts";
import type { Problem } from "./problem.ts";

export type PasteCustomFormats401 = Problem & {
  code?: PasteCustomFormats401Code;
};
