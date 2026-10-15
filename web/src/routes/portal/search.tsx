import { createFileRoute } from "@tanstack/react-router";

import { Page } from "../../shell/page";

export const Route = createFileRoute("/portal/search")({
  staticData: { title: "Search" },
  component: Search,
});

function Search() {
  return (
    <Page>
      <p className="text-body-touch text-ink-2">
        You&apos;ll be able to find something new and ask for it here.
      </p>
    </Page>
  );
}
