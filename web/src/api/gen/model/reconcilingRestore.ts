export interface ReconcilingRestore {
  /** The Backup's name */
  backup: string;
  /** Whether the Backup was restored on a machine other than the one that took it */
  newMachine: boolean;
  /** Whether the old Controller's folders still need finding */
  rehome: boolean;
  /** When Reconciling began */
  startedAt: string;
  /** When the Backup was taken */
  takenAt: string;
}
