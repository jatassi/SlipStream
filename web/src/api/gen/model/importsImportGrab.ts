import type { ImportsImportGrabStage } from "./importsImportGrabStage.ts";

export interface ImportsImportGrab {
  id: string;
  /** The Release's title */
  release: string;
  stage: ImportsImportGrabStage;
}
