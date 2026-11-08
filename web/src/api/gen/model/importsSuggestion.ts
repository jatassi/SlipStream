export interface ImportsSuggestion {
  /** It meets the rules that let a library scan add a Work by itself */
  confident: boolean;
  /** The record's key at its Content Type's source */
  key: string;
  label: string;
  /**
   * The artwork route for its poster, without a size; null when it has none
   * @nullable
   */
  posterUrl: string | null;
  /**
   * The Work, once it is in the library
   * @nullable
   */
  workId: string | null;
  /**
   * Null when the record has none
   * @nullable
   */
  year: number | null;
}
