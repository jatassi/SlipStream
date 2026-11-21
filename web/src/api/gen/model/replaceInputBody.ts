export interface ReplaceInputBody {
  /** Delete every older Backup, copies included, once the Backup of the new passphrase passes its check */
  deleteOlder: boolean;
}
