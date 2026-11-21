import { type QueryClient, useQuery } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { SubtitlesFileSubtitles, TargetsWantedLanguage } from "../api/gen/model";
import {
  countSubtitleTrack,
  getGetSubtitleAttemptQueryKey,
  getGetSubtitleCandidatesQueryKey,
  getGetWorkSubtitlesSuspenseQueryOptions,
  getListSubtitleProvidersSuspenseQueryOptions,
  getSubtitleAttempt,
  getSubtitleCandidates,
  ignoreSubtitleTrack,
  searchSubtitle,
  useGetWorkSubtitlesSuspense,
  useListSubtitleProvidersSuspense,
} from "../api/gen/query/subtitles/subtitles";
import { ensureQuery } from "../api/query-client";
import { toast } from "../ui/toast";
import { searchingWords } from "./sheet-words";

/** The Work page's subtitle reads, so it opens whole. */
export const loadWorkSubtitles = (queryClient: QueryClient, workId: string) =>
  Promise.all([
    ensureQuery(queryClient, getGetWorkSubtitlesSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getListSubtitleProvidersSuspenseQueryOptions()),
  ]);

/** Each file's subtitles by its id, with the providers that name their sources. */
export function useWorkSubtitles(workId: string) {
  const { files } = useGetWorkSubtitlesSuspense(workId).data;
  const { providers } = useListSubtitleProvidersSuspense().data;
  return {
    byFile: new Map<string, SubtitlesFileSubtitles>(files.map((file) => [file.fileId, file])),
    providers,
  };
}

/** One want of one file, as every action's address names it. */
export type WantAt = {
  readonly workId: string;
  readonly fileId: string;
  readonly want: string;
};

/**
 * A file's subtitle actions that answer at once: Search, and counting an
 * embedded track or not. A refusal is a toast with the server's sentence.
 */
async function run(act: () => Promise<unknown>, done?: string) {
  try {
    await act();
    if (done !== undefined) {
      toast(done);
    }
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export function useFileActions(fileId: string) {
  return {
    search: (want: string, wanted: TargetsWantedLanguage) =>
      run(() => searchSubtitle(fileId, want), searchingWords(wanted)),
    ignore: (stream: number) => run(() => ignoreSubtitleTrack(fileId, stream)),
    count: (stream: number) => run(() => countSubtitleTrack(fileId, stream)),
  };
}

// No event names a Manual subtitle search, so its list is read again while providers still answer.
const searchingPollMs = 1000;

export function useCandidates(fileId: string, want: string, searchId: string | undefined) {
  return useQuery({
    queryKey: getGetSubtitleCandidatesQueryKey(fileId, want, searchId ?? ""),
    queryFn: ({ signal }) => getSubtitleCandidates(fileId, want, searchId ?? "", { signal }),
    enabled: searchId !== undefined,
    refetchInterval: (query) => (query.state.data?.state === "searching" ? searchingPollMs : false),
    gcTime: 0,
  });
}

/** A pick's or an upload's step and outcome; subtitles.attempt refreshes it. */
export function useAttempt(fileId: string, attemptId: string | undefined) {
  return useQuery({
    queryKey: getGetSubtitleAttemptQueryKey(fileId, attemptId ?? ""),
    queryFn: ({ signal }) => getSubtitleAttempt(fileId, attemptId ?? "", { signal }),
    enabled: attemptId !== undefined,
    gcTime: 0,
  });
}
