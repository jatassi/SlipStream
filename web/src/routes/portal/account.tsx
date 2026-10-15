import { createFileRoute } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { useSignOut } from "../../shell/account";
import { Page } from "../../shell/page";
import { Button } from "../../ui/button";

export const Route = createFileRoute("/portal/account")({
  staticData: { title: "Account" },
  component: Account,
});

function Account() {
  const { displayName } = useAccount();
  const signOut = useSignOut();
  return (
    <Page>
      <div className="flex flex-col items-start gap-4">
        <p className="text-body-touch text-ink-2">
          You&apos;re signed in as <span className="font-medium text-ink-1">{displayName}</span>.
        </p>
        <Button onClick={signOut}>Sign out</Button>
      </div>
    </Page>
  );
}
