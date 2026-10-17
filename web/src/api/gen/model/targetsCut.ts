import type { TargetsCutOrigin } from "./targetsCutOrigin.ts";

export interface TargetsCut {
  /** The phrases that name it in a title or file name; Standard is read where none is named */
  aliases: string[];
  /**
   * The Content Type that adds it, for origin type
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  key: string;
  name: string;
  /** Built into SlipStream, or added by a Content Type */
  origin: TargetsCutOrigin;
}
