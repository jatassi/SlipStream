import type { MovesPreviewUnit } from "./movesPreviewUnit.ts";

export interface MovesMovePreview {
  /** The Start button's label, such as Move 41 of 43 */
  button: string;
  expiresAt: string;
  /**
   * Works without files the Start homes in their new root
   * @minimum 0
   */
  homed: number;
  /**
   * The units that aren't red, which a Start confirms
   * @minimum 0
   */
  movable: number;
  /** The preview in a sentence, such as Move 41 of 43 movies to den. 2 can't move: The Office and Heat. */
  recommendation: string;
  /** Names the preview for its Start */
  token: string;
  /** @minimum 0 */
  total: number;
  units: MovesPreviewUnit[];
}
