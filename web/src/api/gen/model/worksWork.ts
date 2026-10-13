export interface WorksWork {
  addedAt: string;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  id: string;
  key: string;
  label: string;
  refreshedAt: string;
  sortLabel: string;
}
