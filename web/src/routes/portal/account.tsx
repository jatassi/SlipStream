import { Link, createFileRoute } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { fromPortal } from "../../shell/account";
import { Page } from "../../shell/page";
import { askSignOut } from "../../shell/sign-out";
import { Button } from "../../ui/button";

export const Route = createFileRoute("/portal/account")({
  staticData: { title: "Account" },
  component: Account,
});

function Account() {
  const { displayName, admin } = useAccount();
  return (
    <Page>
      <div className="flex flex-col items-start gap-4">
        <p className="text-body-touch text-ink-2">
          You&apos;re signed in as <span className="font-medium text-ink-1">{displayName}</span>.
        </p>
        <div className="flex flex-wrap gap-2">
          {admin && (
            <Button
              icon={fromPortal.icon}
              nativeButton={false}
              render={<Link to={fromPortal.to} />}
            >
              {fromPortal.label}
            </Button>
          )}
          <Button onClick={askSignOut}>Sign out</Button>
        </div>
      </div>
    </Page>
  );
}
