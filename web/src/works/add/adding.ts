import type { QueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { sentenceOf } from "../../api/errors";
import type { WorksContentType } from "../../api/gen/model";
import { getSearchCatalogSuspenseQueryOptions, useAddWork } from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { toast } from "../../ui/toast";
import { closeAddSheet } from "./sheet-state";

/** Adds a Work and opens it; `searches` says whether the add's search scope searches anything. */
export function useAdding(searches: boolean) {
  const navigate = useNavigate();
  return useAddWork({
    mutation: {
      onSuccess: async ({ created, work }) => {
        closeAddSheet();
        toast(addedWords(work.label, created, searches));
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
      onError: (error) => {
        toast(sentenceOf(error), { kind: "fail" });
      },
    },
  });
}

export function addedWords(label: string, created: boolean, searches: boolean): string {
  if (!created) {
    return `${label} is already in the library.`;
  }
  return searches ? `Added ${label}. Searching now.` : `Added ${label}`;
}

export async function loadCandidates(
  queryClient: QueryClient,
  type: WorksContentType,
  query: string,
) {
  await ensureQuery(queryClient, getSearchCatalogSuspenseQueryOptions(type.id, { query }));
}
