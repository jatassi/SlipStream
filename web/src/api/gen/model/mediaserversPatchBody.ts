export interface MediaserversPatchBody {
  /**
   * @minLength 1
   * @maxLength 2048
   */
  address?: string;
  /**
   * Jellyfin and Emby. Write-only
   * @minLength 1
   * @maxLength 1024
   */
  apiKey?: string;
  /**
   * null for no Open in button
   * @nullable
   */
  browserAddress?: string | null;
  /**
   * null for address
   * @nullable
   */
  fallbackAddress?: string | null;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name?: string;
  reachingNodeId?: string;
  /**
   * A Node id; null when the server runs on no Node
   * @nullable
   */
  runsOnNodeId?: string | null;
}
