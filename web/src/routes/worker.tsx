import { createFileRoute } from "@tanstack/react-router";

import {
  getGetServingSuspenseQueryOptions,
  useGetServingSuspense,
} from "../api/gen/query/serving/serving";
import { ensureQuery } from "../api/query-client";
import { requireEntry } from "../session/gate";
import { EntryPage } from "../shell/entry-page";
import { Data } from "../ui/data";

export const Route = createFileRoute("/worker")({
  beforeLoad: ({ context, location }) => requireEntry(context, "worker", location),
  loader: ({ context }) => ensureQuery(context.queryClient, getGetServingSuspenseQueryOptions()),
  component: Worker,
});

function Worker() {
  const { data } = useGetServingSuspense();
  return (
    <EntryPage title="This Node is a Worker. Manage it from its Controller.">
      <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
        <dt className="text-ink-3">Name</dt>
        <dd className="wrap-anywhere text-ink-1">{data.name}</dd>
        <dt className="text-ink-3">Controller</dt>
        <dd className="wrap-anywhere text-ink-1">{data.controller}</dd>
        <dt className="text-ink-3">Key</dt>
        <dd className="text-ink-1">
          <Data>{data.fingerprint}</Data>
        </dd>
      </dl>
    </EntryPage>
  );
}
