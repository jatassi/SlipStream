export type GetExtensionImpactParams = {
  /**
   * The video file extensions to judge, such as mkv, without the dot
   * @minItems 1
   * @maxItems 100
   * @nullable
   */
  extensions?: string[] | null;
};
