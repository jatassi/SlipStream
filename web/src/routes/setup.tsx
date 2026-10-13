import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

import { requireEntry } from "../session/gate";
import { markClaimed } from "../session/session";
import { Button } from "../ui/button";
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
    <main className="mx-auto flex max-w-sm flex-col gap-6 px-4 py-12">
      <h1 className="text-2xl font-bold">Set up SlipStream</h1>
      {apiKey === undefined ? (
        <>
          <p>
            Enter the Setup code from this Node&apos;s console or log, then create the first Admin.
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
          <ReadOnlyField
            label="API key"
            value={apiKey}
            description="Scripts send this key in the X-Api-Key header. SlipStream shows it only once."
          />
          <Button
            variant="primary"
            onClick={() => {
              void navigate({ to: "/" });
            }}
          >
            Continue
          </Button>
        </>
      )}
    </main>
  );
}
