import type { QueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { getListProfilesSuspenseQueryOptions } from "../api/gen/query/targets/targets";
import { getSearchCatalogSuspenseQueryOptions, useAddWork } from "../api/gen/query/works/works";
import { ensureQuery } from "../api/query-client";
import type { WorksContentType } from "../api/gen/model";
import { toast } from "../ui/toast";

export function useAdding() {
  const navigate = useNavigate();
  return useAddWork({
    mutation: {
      onSuccess: async ({ created, work }) => {
        toast(created ? `Added ${work.label}` : `${work.label} is already in the library`);
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
}

/** The catalog's answer to `query`, and the Profiles the add sheet picks from. */
export async function loadCandidates(
  queryClient: QueryClient,
  type: WorksContentType,
  query: string,
) {
  await Promise.all([
    ensureQuery(queryClient, getSearchCatalogSuspenseQueryOptions(type.id, { query })),
    ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
  ]);
}
