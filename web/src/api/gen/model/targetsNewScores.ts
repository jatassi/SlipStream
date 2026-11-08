import type { TargetsFormatScore } from "./targetsFormatScore.ts";

export interface TargetsNewScores {
  /**
   * A Custom Format left out scores 0
   * @nullable
   */
  formats?: TargetsFormatScore[] | null;
  /**
   * How far a Release's score must pass the file's to replace it in the same tier
   * @minimum 1
   */
  increment?: number;
  /** A Release scoring below it isn't grabbed */
  minimum?: number;
  /** A file at or past the cutoff upgrades to better-scored Releases until it scores this */
  upgradeUntil?: number;
}
