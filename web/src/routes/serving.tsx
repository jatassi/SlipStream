import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import {
  getGetServingSuspenseQueryOptions,
  useGetServingSuspense,
} from "../api/gen/query/serving/serving";
import { ensureQuery } from "../api/query-client";
import { requireEntry } from "../session/gate";
import { statusQuery } from "../session/session";
import { useNow } from "../tasks/use-now";
import { UnreachableCard } from "./-setup/unreachable-card";

const recheckMs = 5000;

export const Route = createFileRoute("/serving")({
  beforeLoad: ({ context, location }) => requireEntry(context, "unreachable", location),
  loader: ({ context }) => ensureQuery(context.queryClient, getGetServingSuspenseQueryOptions()),
  component: Serving,
});

// The Worker's own app shell boots here while its Controller can't be
// reached, and goes back once getStatus, which the Worker sends on to its
// Controller, answers.
function Serving() {
  const { data } = useGetServingSuspense();
  const answered = useQuery({ ...statusQuery, refetchInterval: recheckMs, retry: false }).isSuccess;
  const navigate = useNavigate();
  const now = useNow(recheckMs);
  useEffect(() => {
    if (answered) {
      void navigate({ to: "/", replace: true });
    }
  }, [answered, navigate]);
  return <UnreachableCard data={data} now={now} />;
}
