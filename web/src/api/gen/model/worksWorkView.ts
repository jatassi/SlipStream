import type { WorksArt } from "./worksArt.ts";
import type { WorksFieldValue } from "./worksFieldValue.ts";
import type { WorksItem } from "./worksItem.ts";
import type { WorksLevel } from "./worksLevel.ts";
import type { WorksRating } from "./worksRating.ts";
import type { WorksWebLink } from "./worksWebLink.ts";
import type { WorksWorkProfile } from "./worksWorkProfile.ts";
import type { WorksWorkRoot } from "./worksWorkRoot.ts";

export interface WorksWorkView {
  addedAt: string;
  /** The Work's own images, by kind */
  artwork: WorksArt[];
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  fields: WorksFieldValue[];
  /**
   * When the metadata source last reported that it no longer has this Work; null while it has it
   * @nullable
   */
  goneUpstreamAt: string | null;
  id: string;
  /** By their level's order, live before withdrawn, then position */
  items: WorksItem[];
  key: string;
  label: string;
  /** By depth, then their parent's order, live before withdrawn, then position */
  levels: WorksLevel[];
  /** Pages about the Work on other sites, in its Content Type's order */
  links: WorksWebLink[];
  /** The Profile the Work wants from */
  profile: WorksWorkProfile;
  /** The Work's own, by source */
  ratings: WorksRating[];
  refreshedAt: string;
  /** The root the Work lives in; null while it has none */
  root: WorksWorkRoot | null;
  sortLabel: string;
}
