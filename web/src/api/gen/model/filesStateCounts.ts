export interface FilesStateCounts {
  /** @minimum 1 */
  downloading?: number;
  /** @minimum 1 */
  failed?: number;
  /** @minimum 1 */
  met?: number;
  /** @minimum 1 */
  missing?: number;
  /** @minimum 1 */
  unreleased?: number;
  /** @minimum 1 */
  upgradable?: number;
}
