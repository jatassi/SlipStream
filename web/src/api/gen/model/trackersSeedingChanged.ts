export interface TrackersSeedingChanged {
  /**
   * The Tracker whose seeding changed; null for the default goal, the seeding settings and torrents with no known Tracker
   * @nullable
   */
  trackerId: string | null;
}
