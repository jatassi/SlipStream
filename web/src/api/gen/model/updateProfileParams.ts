import type { UpdateProfileUnwanted } from "./updateProfileUnwanted.ts";

export type UpdateProfileParams = {
  /**
   * The deleted subtitle count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed
   * @minimum 0
   */
  confirmSubtitles?: number;
  /**
   * Return the change's Impact and save nothing
   */
  preview?: boolean;
  /**
   * What happens to the files no Target uses after the change
   */
  unwanted?: UpdateProfileUnwanted;
  /**
   * The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed
   * @minimum 0
   */
  confirm?: number;
  /**
   * The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed
   * @minimum 0
   */
  confirmMade?: number;
};
