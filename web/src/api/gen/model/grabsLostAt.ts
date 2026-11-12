import type { GrabsLostAtStep } from "./grabsLostAtStep.ts";

export interface GrabsLostAt {
  /** This Release's value at the step: a Target's name for want, a quality's name for tier, such as v2 or v1 REPACK for revision, a score, whether another Want chose it for chosen_by_another, a count of the Items it covers that aren't wanted for fewer_unwanted, an Indexer priority, a download factor from 0 to 1, a seeder count, bytes, or the title */
  own: string;
  /** The reference Release's value at the step, written as own is. The reference is the grabbed Release, or the first a search could take when the run grabbed none. */
  ref: string;
  /** The first step of the ranking where the reference Release ranked above this one */
  step: GrabsLostAtStep;
  /**
   * The Target the tier, revision and score steps compare at; null at the other steps and once the Target is deleted
   * @nullable
   */
  targetId: string | null;
}
