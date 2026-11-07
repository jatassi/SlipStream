import type { WorksNewWorkPreset } from "./worksNewWorkPreset.ts";

export interface WorksNewWork {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Add every Item of the aside levels, such as Specials, to what the preset monitors, now and as new ones arrive; left out, the type's add default */
  includeAsides?: boolean;
  /**
   * The Work's key at its metadata source
   * @minLength 1
   * @maxLength 200
   */
  key: string;
  /** Put a new Work's files in a folder per level; left out, the type's add default. Unread for a Content Type with no levels */
  levelFolders?: boolean;
  /** The monitoring preset a new Work's Items follow; left out, the type's add default. Adding a Work already in the library changes nothing */
  preset?: WorksNewWorkPreset;
  /** The Profile of the Content Type a new Work wants from; left out, the type's default */
  profileId?: string;
  /** Make each choice given here its type's default for the next add. Adding a Work already in the library changes no default */
  remember?: boolean;
  /** The root of the Content Type a new Work lives in; left out, the type's default, or none while the type has no root */
  rootId?: string;
}
