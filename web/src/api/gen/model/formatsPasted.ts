import type { FormatsPastedApp } from "./formatsPastedApp.ts";
import type { FormatsPastedFormat } from "./formatsPastedFormat.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export interface FormatsPasted {
  /** The app the formats are read for: the one given, or the one their kinds settle */
  app: FormatsPastedApp;
  /** In the text's order */
  formats: FormatsPastedFormat[];
  impact: TargetsImpact;
}
