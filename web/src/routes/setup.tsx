import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

import { requireEntry } from "../session/gate";
import { markClaimed } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { ReadOnlyField } from "../ui/read-only-field";
import { ClaimForm } from "./-setup/claim-form";

export const Route = createFileRoute("/setup")({
  beforeLoad: ({ context, location }) => requireEntry(context, "setup", location),
  component: Setup,
});

function Setup() {
  const { queryClient } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const [apiKey, setApiKey] = useState<string>();
  return (
    <EntryPage title="Set up SlipStream">
      {apiKey === undefined ? (
        <>
          <p>
            Create the first Admin for this Node. The Setup code is in this Node&apos;s console and
            log.
          </p>
          <ClaimForm
            onClaimed={(claimed) => {
              markClaimed(queryClient, claimed);
              setApiKey(claimed.apiKey);
            }}
            onAlreadyClaimed={() => {
              void navigate({ to: "/sign-in" });
            }}
          />
        </>
      ) : (
        <>
          <p className="text-ink-1">SlipStream is ready, and you&apos;re its first Admin.</p>
          <ReadOnlyField
            label="API key"
            value={apiKey}
            description="SlipStream shows this key only once."
          />
          <Details summary="Copy it now if scripts or other apps will use SlipStream.">
            Scripts send the key in the X-Api-Key header of each request.
          </Details>
          <Button
            look="primary"
            onClick={() => {
              void navigate({ to: "/" });
            }}
          >
            Continue
          </Button>
        </>
      )}
    </EntryPage>
  );
}
