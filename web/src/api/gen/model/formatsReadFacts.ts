import type { FormatsReadFactsIds } from "./formatsReadFactsIds.ts";
import type { FormatsReadFactsSpan } from "./formatsReadFactsSpan.ts";
import type { FormatsRevision } from "./formatsRevision.ts";

export interface FormatsReadFacts {
  /** Audio codecs, best first */
  audio: string[];
  /** Channel layouts, best first */
  channels: string[];
  /** The cut's key, standard when the name names none */
  cut: string;
  /**
   * The edition text as the name gives it
   * @nullable
   */
  edition: string | null;
  /**
   * The release group
   * @nullable
   */
  group: string | null;
  /** Best first */
  hdr: string[];
  /** The id tags the name carries, by Torznab's names: tmdbid, imdbid, tvdbid */
  ids: FormatsReadFactsIds;
  /** ISO 639 codes */
  languages: string[];
  /** What the name numbers, such as S01E02, in words */
  marks: string[];
  /** @nullable */
  modifier: string | null;
  /**
   * The Quality key, such as webdl-1080p; null when the name gives none
   * @nullable
   */
  quality: string | null;
  /** @nullable */
  resolution: string | null;
  revision: FormatsRevision;
  /**
   * The fine source, such as webdl
   * @nullable
   */
  source: string | null;
  /** One Item, several Items of one level, a whole level or several levels; empty when the name numbers nothing */
  span: FormatsReadFactsSpan;
  /** Every title the name gives, as in A AKA B */
  titles: string[];
  /**
   * The video codec
   * @nullable
   */
  video: string | null;
  /** @nullable */
  year: number | null;
}
