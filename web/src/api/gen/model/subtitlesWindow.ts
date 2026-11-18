export interface SubtitlesWindow {
  /** Whether the provider counts the window up to the last call, or reports it with a lag */
  exact: boolean;
  /** @minimum 0 */
  limit: number;
  /** @minimum 0 */
  remaining: number;
  resets: string;
  /** @minimum 0 */
  used: number;
}
