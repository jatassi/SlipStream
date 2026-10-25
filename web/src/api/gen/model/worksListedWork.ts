import type { WorksArt } from "./worksArt.ts";
import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksListedWork {
  addedAt: string;
  /** The Work's own images, by kind */
  artwork: WorksArt[];
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** The Work's own fields */
  fields: WorksFieldValue[];
  /**
   * When the metadata source last reported that it no longer has this Work; null while it has it
   * @nullable
   */
  goneUpstreamAt: string | null;
  id: string;
  key: string;
  label: string;
  refreshedAt: string;
  sortLabel: string;
}
