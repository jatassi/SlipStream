import type { InvalidateQueryFilters } from "@tanstack/react-query";

import {
  getGetSubtitleAttemptQueryKey,
  getGetSubtitleSettingsQueryKey,
  getGetWorkSubtitlesQueryKey,
  getListSubtitleProvidersQueryKey,
} from "../api/gen/query/subtitles/subtitles";
import { everyPath } from "../live/refresh";

export const subtitleSettings: InvalidateQueryFilters = {
  queryKey: getGetSubtitleSettingsQueryKey(),
};

export const subtitleProviders: InvalidateQueryFilters = {
  queryKey: getListSubtitleProvidersQueryKey(),
};

export const everyNodeSettings = everyPath(/^\/api\/v1\/subtitles\/nodes\/[^/]+\/settings$/u);

export const workSubtitles = (workId: string): InvalidateQueryFilters => ({
  queryKey: getGetWorkSubtitlesQueryKey(workId),
});

export const subtitleAttempt = (fileId: string, attemptId: string): InvalidateQueryFilters => ({
  queryKey: getGetSubtitleAttemptQueryKey(fileId, attemptId),
});
