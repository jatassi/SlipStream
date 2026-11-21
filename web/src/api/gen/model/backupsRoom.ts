export interface BackupsRoom {
  /** The bytes SlipStream may write on the folder's volume */
  free: number;
  /** The bytes of every listed Backup */
  need: number;
}
