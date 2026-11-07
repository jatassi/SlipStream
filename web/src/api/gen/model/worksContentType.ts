import type { WorksArtRoles } from "./worksArtRoles.ts";
import type { WorksArtworkKind } from "./worksArtworkKind.ts";
import type { WorksContentTypeIcon } from "./worksContentTypeIcon.ts";
import type { WorksContentTypeItems } from "./worksContentTypeItems.ts";
import type { WorksFieldSpec } from "./worksFieldSpec.ts";
import type { WorksNoun } from "./worksNoun.ts";

export interface WorksContentType {
  art: WorksArtRoles;
  /** The kinds of image its Works and Items carry */
  artwork: WorksArtworkKind[];
  enabled: boolean;
  fields: WorksFieldSpec[];
  icon: WorksContentTypeIcon;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  id: string;
  item: WorksNoun;
  /** How many Items a Work holds, which sets the Work page's layout */
  items: WorksContentTypeItems;
  /** Outermost first; empty when Items hang off the Work */
  levels: WorksNoun[];
  name: string;
  /** The ratings sources its Works show, in order. The first a Work holds is its hero rating */
  ratings: string[];
  /** Its metadata source's name, such as TMDB */
  source: string;
  work: WorksNoun;
}
