import type { WorksArt } from "./worksArt.ts";
import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksLevel {
  /** By kind */
  artwork: WorksArt[];
  /** An outermost level outside the Work's main run, such as Specials */
  aside: boolean;
  /** @minimum 1 */
  depth: number;
  fields: WorksFieldValue[];
  id: string;
  key: string;
  label: string;
  /**
   * Null under the Work
   * @nullable
   */
  parentId: string | null;
  /** @minimum 1 */
  position: number;
  /**
   * When the source stopped listing it; null while it lists it
   * @nullable
   */
  withdrawnAt: string | null;
}
