import { infiniteQueryOptions } from "@tanstack/react-query";

import type { ReconcilingGroup } from "../api/gen/model";
import {
  getListReconcilingTorrentsQueryKey,
  listReconcilingTorrents,
} from "../api/gen/query/reconciling/reconciling";

const reviewPage = 50;

type Group = Pick<ReconcilingGroup, "reason" | "found">;

// The key extends the generated one, so reconciling.changed refreshes every page loaded.
export const groupQuery = ({ reason, found }: Group) =>
  infiniteQueryOptions({
    queryKey: [
      ...getListReconcilingTorrentsQueryKey({ reason, found, limit: reviewPage }),
      "pages",
    ],
    queryFn: ({ pageParam, signal }) =>
      listReconcilingTorrents(
        { reason, found, limit: reviewPage, ...(pageParam !== "" && { after: pageParam }) },
        { signal },
      ),
    initialPageParam: "",
    getNextPageParam: (page) => page.next ?? undefined,
  });
