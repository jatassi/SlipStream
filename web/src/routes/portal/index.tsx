import { createFileRoute } from "@tanstack/react-router";

import { Page } from "../../shell/page";

export const Route = createFileRoute("/portal/")({
  staticData: { title: "Home" },
  component: Home,
});

function Home() {
  return (
    <Page>
      <p className="text-body-touch text-ink-2">
        New arrivals and the things you&apos;ve asked for will show here.
      </p>
    </Page>
  );
}
