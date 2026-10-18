export interface TargetsEstimate {
  /**
   * That rate over the runtime of the Profile's Works, null with no estimate yet
   * @minimum 0
   * @nullable
   */
  bytes: number | null;
  /**
   * What the files filling the Target average, null with no estimate yet
   * @minimum 0
   * @nullable
   */
  mibPerMinute: number | null;
  targetId: string;
}
