import { createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";
import { EntryPage } from "../shell/entry-page";

export const Route = createFileRoute("/worker")({
  beforeLoad: ({ context, location }) => requireEntry(context, "worker", location),
  component: Worker,
});

function Worker() {
  return <EntryPage title="This Node is a Worker. Manage it from its Controller." />;
}
