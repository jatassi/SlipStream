export interface WorksPerson {
  /** Empty when the source names none */
  department: string;
  name: string;
  /**
   * The artwork route for the person's photo, without a size; null when the source has none
   * @nullable
   */
  photoUrl: string | null;
  /** The character played or the job done; empty when the source names none */
  role: string;
}
