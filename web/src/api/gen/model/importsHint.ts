import type { ImportsHintID } from "./importsHintID.ts";

export interface ImportsHint {
  /** The provider ids the name's tags carry */
  ids: ImportsHintID[];
  /** The title the folder or file name carries, as written */
  title: string;
  /**
   * Null when the name carries none
   * @nullable
   */
  year: number | null;
}
