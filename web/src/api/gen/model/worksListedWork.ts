import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksListedWork {
  addedAt: string;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** The Work's own fields */
  fields: WorksFieldValue[];
  id: string;
  key: string;
  label: string;
  refreshedAt: string;
  sortLabel: string;
}
