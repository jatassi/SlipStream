import { queueRows, wholeLibrary } from "../live/rows";
import type { LiveEffects } from "../live/types";
import {
  everyNodeSettings,
  subtitleAttempt,
  subtitleProviders,
  subtitleSettings,
  workSubtitles,
} from "./keys";

// Past this many Works a change reads the whole Library once instead of row by row.
const widestSplice = 1000;

/**
 * A change to what files want or have, to the Subtitle Providers and their settings, or to one
 * subtitle download or upload being checked.
 */
export const subtitleEffects = {
  "subtitles.changed": ({ workIds }, { queryClient, refresh }) => {
    if (workIds.length > widestSplice) {
      refresh(wholeLibrary);
    } else {
      queueRows(queryClient, workIds);
    }
    for (const workId of workIds) {
      refresh(workSubtitles(workId));
    }
    refresh(subtitleSettings);
  },
  "subtitles.providers-changed": (_data, { refresh }) => {
    refresh(subtitleProviders);
    refresh(subtitleSettings);
    refresh(everyNodeSettings);
  },
  "subtitles.attempt": ({ fileId, attemptId }, { refresh }) => {
    refresh(subtitleAttempt(fileId, attemptId));
  },
} satisfies Pick<
  LiveEffects,
  "subtitles.attempt" | "subtitles.changed" | "subtitles.providers-changed"
>;
