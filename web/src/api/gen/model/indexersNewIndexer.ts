export interface IndexersNewIndexer {
  /** @maxLength 256 */
  apiKey: string;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /**
   * The Torznab feed's base; SlipStream adds /api
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?$
   */
  url: string;
}
