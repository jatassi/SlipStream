export interface PlaybackVersion {
  fileId: string;
  label: string;
  /** The server lists the file as a separate item, which the player can't choose as a version */
  separateItem: boolean;
}
