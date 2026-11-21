export interface ReconcilingRootRow {
  /** Whether Reconciling stopped waiting on its scan */
  continued: boolean;
  id: string;
  /** The root as a person reads it, such as Movies library root on Media server */
  label: string;
  /** Whether the re-home step saved it. A root not saved waits for its folder, and no scan reads it */
  saved: boolean;
  /** Whether a scan finished since it was saved */
  scanned: boolean;
}
