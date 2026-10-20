import type { WorksArtworkKind } from "./worksArtworkKind.ts";
import type { WorksContentTypeIcon } from "./worksContentTypeIcon.ts";
import type { WorksFieldSpec } from "./worksFieldSpec.ts";
import type { WorksNoun } from "./worksNoun.ts";

export interface WorksContentType {
  /** The kinds of image its Works and Items carry */
  artwork: WorksArtworkKind[];
  enabled: boolean;
  fields: WorksFieldSpec[];
  icon: WorksContentTypeIcon;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  id: string;
  item: WorksNoun;
  /** Outermost first; empty when Items hang off the Work */
  levels: WorksNoun[];
  name: string;
  work: WorksNoun;
}
