import type { SubtitlesCandidate } from "../api/gen/model";
import {
  type SubtitlesWorld,
  missing,
  openOrchard,
  subtitleRow,
  uhdSubtitles,
} from "./subtitles-work";

/** A SubDL candidate for The Quiet Orchard whose release ends in key. */
export const candidate = (
  key: string,
  change: Partial<SubtitlesCandidate> = {},
): SubtitlesCandidate => ({
  key,
  provider: "subdl",
  release: `The.Quiet.Orchard.2021.2160p.${key}`,
  reasons: [],
  disabled: false,
  tried: null,
  ...change,
});

/** The Quiet Orchard's Work page with its English subtitle missing, over this subtitles world. */
export const openMissing = (world: Omit<SubtitlesWorld, "work">) =>
  openOrchard({ work: { files: [uhdSubtitles([missing("en", "nothing_found")])] }, ...world });

/** The English row's Manual search on desktop. */
export const manualSearch = () =>
  subtitleRow("English").getByRole("button", { name: "Manual search" });
