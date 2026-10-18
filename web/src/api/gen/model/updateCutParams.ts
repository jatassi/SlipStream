import type { UpdateCutUnwanted } from "./updateCutUnwanted.ts";

export type UpdateCutParams = {
  /**
   * Answer the change's Impact and save nothing
   */
  preview?: boolean;
  /**
   * What becomes of the files the change leaves filling nothing, and of downloads in flight for cells it removes
   */
  unwanted?: UpdateCutUnwanted;
  /**
   * The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed
   * @minimum 0
   */
  confirm?: number;
};
