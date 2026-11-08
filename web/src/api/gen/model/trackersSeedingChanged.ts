export interface TrackersSeedingChanged {
  /**
   * The Tracker whose seeding changed; null for the default goal and the seeding settings
   * @nullable
   */
  trackerId: string | null;
}
