import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksItem {
  fields: WorksFieldValue[];
  id: string;
  key: string;
  label: string;
  /**
   * Null when the Content Type declares no levels
   * @nullable
   */
  levelId: string | null;
  monitored: boolean;
  /** @minimum 1 */
  position: number;
  /**
   * When the source stopped listing it; null while it lists it
   * @nullable
   */
  withdrawnAt: string | null;
}
