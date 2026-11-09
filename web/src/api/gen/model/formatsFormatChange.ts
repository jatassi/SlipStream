import type { FormatsCustomFormat } from "./formatsCustomFormat.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export interface FormatsFormatChange {
  format: FormatsCustomFormat;
  impact: TargetsImpact;
}
