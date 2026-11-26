import type { PlaybackEvidenceFact } from "./playbackEvidenceFact.ts";

export interface PlaybackEvidence {
  contentType: string;
  fact: PlaybackEvidenceFact;
  playId: string;
  startedAt: string;
  title: string;
  value: string;
  workId: string;
}
