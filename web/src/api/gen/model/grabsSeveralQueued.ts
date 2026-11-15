import type { GrabsWorkOutcome } from "./grabsWorkOutcome.ts";

export interface GrabsSeveralQueued {
  /**
   * Works left out because only gone files want a Release
   * @minimum 0
   */
  leftGone: number;
  /**
   * Works whose every group was already queued or searching
   * @minimum 0
   */
  merged: number;
  /**
   * Works with at least one group newly queued
   * @minimum 0
   */
  queued: number;
  /** One per Work, in request order */
  works: GrabsWorkOutcome[];
}
