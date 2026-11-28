import type { ConversionsAttempt } from "./conversionsAttempt.ts";
import type { ConversionsConversionRowKind } from "./conversionsConversionRowKind.ts";
import type { ConversionsConversionRowLane } from "./conversionsConversionRowLane.ts";
import type { ConversionsConversionRowStage } from "./conversionsConversionRowStage.ts";
import type { ConversionsFailure } from "./conversionsFailure.ts";
import type { ConversionsItem } from "./conversionsItem.ts";
import type { ConversionsNamed } from "./conversionsNamed.ts";
import type { ConversionsProgress } from "./conversionsProgress.ts";
import type { ConversionsSource } from "./conversionsSource.ts";
import type { ConversionsTarget } from "./conversionsTarget.ts";
import type { ConversionsWait } from "./conversionsWait.ts";
import type { ConversionsWork } from "./conversionsWork.ts";

export interface ConversionsConversionRow {
  /** Oldest first */
  attempts: ConversionsAttempt[];
  /** Why it failed; null unless it failed */
  failure: ConversionsFailure | null;
  /**
   * When it is forecast to finish; null with no forecast
   * @nullable
   */
  finish: string | null;
  id: string;
  /** In tree order */
  items: ConversionsItem[];
  kind: ConversionsConversionRowKind;
  lane: ConversionsConversionRowLane;
  /** The Node it is placed on; null until it is placed */
  node: ConversionsNamed | null;
  /**
   * The Media Servers whose transcodes pause it; null while it isn't paused
   * @nullable
   */
  pausedBy: string[] | null;
  /** While it runs; conversions.live keeps it current. null otherwise */
  progress: ConversionsProgress | null;
  source: ConversionsSource;
  stage: ConversionsConversionRowStage;
  /**
   * When it is forecast to start; null once it has, or with no forecast
   * @nullable
   */
  start: string | null;
  /** The Target it makes a file for */
  target: ConversionsTarget;
  /** Why it hasn't started; null once it has */
  wait: ConversionsWait | null;
  work: ConversionsWork;
}
