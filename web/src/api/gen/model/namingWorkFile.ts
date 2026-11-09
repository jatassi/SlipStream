export interface NamingWorkFile {
  /** Why its name doesn't fit, then what to do */
  detail?: string;
  /** Where the file is now, relative to its root */
  from: string;
  /** Where the draft puts it; empty when its name doesn't fit */
  to: string;
}
