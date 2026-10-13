import { createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";

export const Route = createFileRoute("/worker")({
  beforeLoad: ({ context, location }) => requireEntry(context, "worker", location),
  component: Worker,
});

function Worker() {
  return (
    <main className="mx-auto flex max-w-sm flex-col gap-6 px-4 py-12">
      <h1 className="text-2xl font-bold">This Node is a Worker. Manage it from its Controller.</h1>
    </main>
  );
}
