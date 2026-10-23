export interface HealthUnreached {
  /** The item it was for; absent for a whole category */
  key?: string;
  /** What couldn't be reached, as a sentence */
  words: string;
}
