export interface WorksNewWork {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * The Work's key at its metadata source
   * @minLength 1
   * @maxLength 200
   */
  key: string;
  /** Monitor a new Work's Items; adding a Work already in the library changes nothing */
  monitor?: boolean;
  /** The Profile of the Content Type a new Work wants from; left out, the type's default */
  profileId?: string;
  /** The root of the Content Type a new Work lives in; left out, the type's default, or none while the type has no root */
  rootId?: string;
}
