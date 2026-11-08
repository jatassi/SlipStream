export interface FilesSubtitleTrack {
  /** ffprobe's codec name, such as subrip */
  codec: string;
  default: boolean;
  forced: boolean;
  /** A bitmap format, such as PGS */
  image: boolean;
  /**
   * An ISO 639 code; null when the track names none
   * @nullable
   */
  language: string | null;
}
