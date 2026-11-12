import { useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { GrabsGrabPick, GrabsPickResult } from "../api/gen/model";
import { useGrabFromManualSearch } from "../api/gen/query/grabs/grabs";
import type { Results } from "./picks";

/** A pick and where its result shows: a row's pick, or a Release's key in All Releases. */
export type Sent = { readonly key: string; readonly pick: GrabsGrabPick };

/**
 * Grabs picks from one Manual search and keeps each pick's result by its
 * key. A search another Admin replaced refuses the whole request.
 */
export function useGrabbing(workId: string, manualSearchId: string) {
  const [results, setResults] = useState<Results>(new Map());
  const grab = useGrabFromManualSearch();
  return {
    results,
    busy: grab.isPending,
    failure: grab.error === null ? undefined : sentenceOf(grab.error),
    replaced: isProblem(grab.error, "manual_search_replaced"),
    reset: grab.reset,
    post: (sent: readonly Sent[], onAnswer?: (results: readonly GrabsPickResult[]) => void) => {
      grab.mutate(
        { workId, data: { manualSearchId, picks: sent.map(({ pick }) => pick) } },
        {
          onSuccess: (answer) => {
            setResults(
              (before) =>
                new Map([
                  ...before,
                  ...answer.results.flatMap((result, at) => {
                    const key = sent[at]?.key;
                    return key === undefined ? [] : [[key, result] as const];
                  }),
                ]),
            );
            onAnswer?.(answer.results);
          },
        },
      );
    },
  };
}

export type Grabbing = ReturnType<typeof useGrabbing>;
