import { createFileRoute } from "@tanstack/react-router";

import {
  getGetServingSuspenseQueryOptions,
  useGetServingSuspense,
} from "../api/gen/query/serving/serving";
import { ensureQuery } from "../api/query-client";
import { requireEntry } from "../session/gate";
import { EntryPage } from "../shell/entry-page";
import { Data } from "../ui/data";
import { Fact, Facts } from "../ui/facts";

export const Route = createFileRoute("/worker")({
  beforeLoad: ({ context, location }) => requireEntry(context, "worker", location),
  loader: ({ context }) => ensureQuery(context.queryClient, getGetServingSuspenseQueryOptions()),
  component: Worker,
});

function Worker() {
  const { data } = useGetServingSuspense();
  return (
    <EntryPage title="This Node is a Worker. Manage it from its Controller.">
      <Facts look="well">
        <Fact label="Name">{data.name}</Fact>
        <Fact label="Controller">{data.controller}</Fact>
        <Fact label="Key">
          <Data>{data.fingerprint}</Data>
        </Fact>
      </Facts>
    </EntryPage>
  );
}
