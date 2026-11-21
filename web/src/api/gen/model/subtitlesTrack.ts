export interface SubtitlesTrack {
  codec: string;
  forced: boolean;
  /** The Admin chose not to count it */
  ignored: boolean;
  image: boolean;
  language: string;
  /**
   * The want it meets
   * @nullable
   */
  meets: string | null;
  /**
   * Its position among the file's subtitle tracks
   * @minimum 0
   */
  stream: number;
}
