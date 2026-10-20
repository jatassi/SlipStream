import type { WorksArtworkKindShape } from "./worksArtworkKindShape.ts";
import type { WorksArtworkKindTier } from "./worksArtworkKindTier.ts";

export interface WorksArtworkKind {
  key: string;
  label: string;
  shape: WorksArtworkKindShape;
  tier: WorksArtworkKindTier;
}
