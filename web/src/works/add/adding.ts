import type { QueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { sentenceOf } from "../../api/errors";
import type { WorksContentType } from "../../api/gen/model";
import { getSearchCatalogSuspenseQueryOptions, useAddWork } from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { toast } from "../../ui/toast";
import { closeAddSheet } from "./sheet-state";

export function useAdding() {
  const navigate = useNavigate();
  return useAddWork({
    mutation: {
      onSuccess: async ({ created, work }) => {
        closeAddSheet();
        toast(created ? `Added ${work.label}` : `${work.label} is already in the library.`);
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
      onError: (error) => {
        toast(sentenceOf(error), { kind: "fail" });
      },
    },
  });
}

export async function loadCandidates(
  queryClient: QueryClient,
  type: WorksContentType,
  query: string,
) {
  await ensureQuery(queryClient, getSearchCatalogSuspenseQueryOptions(type.id, { query }));
}
