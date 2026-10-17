import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import {
  getGetServingSuspenseQueryOptions,
  useGetServingSuspense,
} from "../api/gen/query/serving/serving";
import { ensureQuery } from "../api/query-client";
import { countWords, sinceWords } from "../format";
import { requireEntry } from "../session/gate";
import { statusQuery } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { useNow } from "../tasks/use-now";

const recheckMs = 5000;

const torrents = { one: "torrent", many: "torrents" } as const;

export const Route = createFileRoute("/serving")({
  beforeLoad: ({ context, location }) => requireEntry(context, "unreachable", location),
  loader: ({ context }) => ensureQuery(context.queryClient, getGetServingSuspenseQueryOptions()),
  component: Serving,
});

function andList(words: readonly string[]): string {
  return words.length < 2
    ? words.join("")
    : `${words.slice(0, -1).join(", ")} and ${words.slice(-1).join("")}`;
}

// Anyone who reaches the Worker may see the card, so it counts and never
// names what anyone requested.
function holdingLine(holding: readonly string[], seeding: number): string {
  const seeds = countWords(seeding, torrents);
  return holding.length === 0
    ? `Seeding ${seeds}.`
    : `Holding ${andList(holding)} · seeding ${seeds}.`;
}

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
  const { since } = data;
  const when = since === undefined ? "" : ` since ${sinceWords(since, { now, zone: undefined })}`;
  return (
    <EntryPage title={`${data.controller} can't be reached from ${data.name}${when}.`}>
      <p>{`${data.name} keeps seeding and finishes what it started.`}</p>
      <p>{holdingLine(data.holding, data.seeding)}</p>
    </EntryPage>
  );
}
