import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { isProblem } from "../api/errors";
import { useConfirmEmail } from "../api/gen/query/identity/identity";
import { requireEntry } from "../session/gate";
import { refreshSession } from "../session/session";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { type LiveProps, LinkPage } from "./-links/link-page";

export const Route = createFileRoute("/confirm-email")({
  beforeLoad: ({ context, location }) => requireEntry(context, "link", location),
  component: ConfirmEmailPage,
});

function useConfirm({ token, linkDied }: Pick<LiveProps, "token" | "linkDied">) {
  const queryClient = useQueryClient();
  const confirm = useConfirmEmail();
  const [failure, setFailure] = useState<string>();
  const [confirmed, setConfirmed] = useState(false);
  const run = async () => {
    setFailure(undefined);
    try {
      await confirm.mutateAsync({ token });
    } catch (error) {
      if (isProblem(error, "link_dead")) {
        linkDied();
      } else {
        setFailure(error instanceof Error ? error.message : String(error));
      }
      return;
    }
    await refreshSession(queryClient);
    setConfirmed(true);
  };
  return { busy: confirm.isPending, failure, confirmed, run };
}

function Confirm({ token, link, linkDied }: LiveProps) {
  const navigate = useNavigate();
  const { busy, failure, confirmed, run } = useConfirm({ token, linkDied });
  if (confirmed) {
    return (
      <>
        <output className="block text-ink-1">Your SlipStream email is now {link.email}.</output>
        <Button
          look="primary"
          onClick={() => {
            void navigate({ to: "/", replace: true });
          }}
        >
          Open SlipStream
        </Button>
      </>
    );
  }
  return (
    <>
      <p>
        Change your {link.serverName} email to <span className="text-ink-1">{link.email}</span>?
        SlipStream emails your old address about the change.
      </p>
      {failure !== undefined && <AlertLine text={failure} />}
      <Button look="primary" busy={busy} onClick={() => void run()}>
        Confirm
      </Button>
    </>
  );
}

function ConfirmEmailPage() {
  return <LinkPage kind="email" title="Confirm your email" forSomeoneNew={false} Live={Confirm} />;
}
