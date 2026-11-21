import type { SubtitlesProvider } from "./subtitlesProvider.ts";

export interface SubtitlesProviderList {
  /** In search order */
  providers: SubtitlesProvider[];
}
