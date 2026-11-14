import type { UpdateTargetUnwanted } from "./updateTargetUnwanted.ts";

export type UpdateTargetParams = {
  /**
   * Return the change's Impact and save nothing
   */
  preview?: boolean;
  /**
   * What becomes of the files the change leaves filling nothing
   */
  unwanted?: UpdateTargetUnwanted;
  /**
   * The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed
   * @minimum 0
   */
  confirm?: number;
};
