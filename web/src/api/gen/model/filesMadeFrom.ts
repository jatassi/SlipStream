export interface FilesMadeFrom {
  /**
   * null once the file is gone
   * @nullable
   */
  fileId: string | null;
  /** The file's name, while it is one of this Work's; empty otherwise */
  name: string;
}
