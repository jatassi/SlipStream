import type { PasteCustomFormats409Code } from "./pasteCustomFormats409Code.ts";
import type { Problem } from "./problem.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export type PasteCustomFormats409 = Problem & {
  code?: PasteCustomFormats409Code;
  impact?: TargetsImpact;
};
