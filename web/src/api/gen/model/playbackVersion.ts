export interface PlaybackVersion {
  fileId: string;
  label: string;
  /** The server lists it as its own item, not as a version the player can choose */
  separateItem: boolean;
}
