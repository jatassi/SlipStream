import { expect, test } from "vitest";

import type { PlaybackBlocked } from "../../api/gen/model";
import { causeWords } from "./playback-words";

const blocked = (dimension: PlaybackBlocked["dimension"], value: string): PlaybackBlocked => ({
  dimension,
  value,
  choosable: true,
});

test.each([
  {
    cause: [blocked("hdr", "DV"), blocked("audio_codec", "TrueHD")],
    words: "the player can't show Dolby Vision or play TrueHD audio",
  },
  {
    cause: [blocked("hdr", "DV"), blocked("hdr", "HDR10"), blocked("channels", "7.1")],
    words: "the player can't show Dolby Vision or HDR10 or play 7.1 audio",
  },
  {
    cause: [blocked("audio_track", "AudioIsExternal"), blocked("container", "MKV")],
    words: "its audio is in a separate file or the player can't play the MKV container",
  },
])("causes join as $words", ({ cause, words }) => {
  expect(causeWords(cause)).toBe(words);
});
