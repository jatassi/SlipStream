import { createFileRoute, useLocation } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { requireEntry } from "../session/gate";
import { markClaimed } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { ClaimForm } from "./-setup/claim-form";

export const Route = createFileRoute("/setup")({
  beforeLoad: ({ context, location }) => requireEntry(context, "setup", location),
  component: Setup,
});

function codeInFragment(hash: string): string | undefined {
  const code = new URLSearchParams(hash).get("code");
  return code === null || code.trim() === "" ? undefined : code;
}

function useLinkedSetupCode() {
  const navigate = Route.useNavigate();
  const hash = useLocation({ select: (location) => location.hash });
  const [linked] = useState(() => codeInFragment(hash));
  useEffect(() => {
    if (linked !== undefined) {
      void navigate({ to: ".", hash: "", replace: true });
    }
  }, [linked, navigate]);
  return linked;
}

function Setup() {
  const { queryClient } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const linkedSetupCode = useLinkedSetupCode();
  return (
    <EntryPage title="Set up SlipStream">
      <p>
        Create the first Admin for this Node. The Setup code is in this Node&apos;s console and log.
      </p>
      <ClaimForm
        linkedSetupCode={linkedSetupCode}
        onClaimed={(signedIn) => {
          markClaimed(queryClient, signedIn);
          void navigate({ to: "/" });
        }}
        onAlreadyClaimed={() => {
          void navigate({ to: "/sign-in" });
        }}
      />
    </EntryPage>
  );
}
