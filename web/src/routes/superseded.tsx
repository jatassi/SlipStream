import { createFileRoute } from "@tanstack/react-router";

import { getGetSupersededSuspenseQueryOptions } from "../api/gen/query/superseded/superseded";
import { ensureQuery } from "../api/query-client";
import { type Entry, requireEntry } from "../session/gate";
import { SupersededPage, type Viewer } from "../superseded/page";

const viewerOf = (entry: Entry): Viewer => {
  if (entry.kind !== "superseded" || entry.account === null) {
    return "signed-out";
  }
  return entry.account.admin ? "admin" : "member";
};

export const Route = createFileRoute("/superseded")({
  beforeLoad: async ({ context, location }) => {
    const viewer = viewerOf(await requireEntry(context, "superseded", location));
    if (viewer === "admin") {
      await context.live.connect();
    }
    return { viewer };
  },
  loader: ({ context }) =>
    context.viewer === "admin"
      ? ensureQuery(context.queryClient, getGetSupersededSuspenseQueryOptions())
      : undefined,
  component: Superseded,
});

function Superseded() {
  const { viewer } = Route.useRouteContext();
  return <SupersededPage viewer={viewer} />;
}
