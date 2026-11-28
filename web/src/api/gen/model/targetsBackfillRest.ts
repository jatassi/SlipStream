export interface TargetsBackfillRest {
  /**
   * The name of a new Profile, the old one without the made-from Target, that the rest move to
   * @maxLength 100
   */
  newName?: string;
  /** An existing Profile the rest move to */
  profileId?: string;
}
