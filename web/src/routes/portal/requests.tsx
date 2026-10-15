import { createFileRoute } from "@tanstack/react-router";

import { Page } from "../../shell/page";

export const Route = createFileRoute("/portal/requests")({
  staticData: { title: "Requests" },
  component: Requests,
});

function Requests() {
  return (
    <Page>
      <p className="text-body-touch text-ink-2">
        Everything you&apos;ve asked for, and how close each one is, will show here.
      </p>
    </Page>
  );
}
