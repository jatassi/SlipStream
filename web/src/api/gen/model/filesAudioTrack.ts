export interface FilesAudioTrack {
  atmos: boolean;
  /** @minimum 0 */
  channels: number;
  /** ffprobe's codec name, such as eac3 */
  codec: string;
  default: boolean;
  /**
   * An ISO 639 code; null when the track names none
   * @nullable
   */
  language: string | null;
  /** ffprobe's channel layout, such as 5.1(side); empty when it names none */
  layout: string;
  /** Such as Dolby Digital Plus + Dolby Atmos; empty when ffprobe names none */
  profile: string;
  /** Empty when the track has none */
  title: string;
}
