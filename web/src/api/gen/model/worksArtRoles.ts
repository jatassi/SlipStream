export interface WorksArtRoles {
  /**
   * The kind shown behind the Work page's header; null when none
   * @nullable
   */
  backdrop: string | null;
  /**
   * The kind the Library and the Work page show as its poster; null when none
   * @nullable
   */
  poster: string | null;
  /**
   * The kind of logo shown as the Work's title; null when none
   * @nullable
   */
  title: string | null;
}
