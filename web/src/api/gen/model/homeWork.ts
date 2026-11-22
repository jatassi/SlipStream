export interface HomeWork {
  /**
   * The artwork route without a size; null when it has none
   * @nullable
   */
  backdropUrl: string | null;
  contentType: string;
  /** Its status is failed, so it never tints Home */
  failed: boolean;
  id: string;
  label: string;
  /**
   * The artwork route without a size; null when it has none
   * @nullable
   */
  posterUrl: string | null;
}
