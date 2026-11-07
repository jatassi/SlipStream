import type { WorksArtworkKindShape } from "./worksArtworkKindShape.ts";
import type { WorksArtworkKindTier } from "./worksArtworkKindTier.ts";

export interface WorksArtworkKind {
  /**
   * The level's depth, 1 for the outermost; present only on tier level
   * @minimum 1
   */
  depth?: number;
  key: string;
  label: string;
  shape: WorksArtworkKindShape;
  tier: WorksArtworkKindTier;
}
