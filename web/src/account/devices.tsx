import { useState } from "react";

import type { IdentitySessionRow } from "../api/gen/model";
import {
  useEndOtherSessions,
  useEndSession,
  useListSessionsSuspense,
} from "../api/gen/query/identity/identity";
import { clock } from "../format";
import { askSignOut } from "../shell/sign-out";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";

type Ending =
  | { readonly kind: "one"; readonly session: IdentitySessionRow }
  | { readonly kind: "others" }
  | null;

const metaOf = ({ address, current, lastSeenAt }: IdentitySessionRow): string =>
  current ? `${address} · This device` : `${address} · Active ${clock(lastSeenAt)}`;

function DeviceRow({
  session,
  end,
}: {
  session: IdentitySessionRow;
  end: (ending: Ending) => void;
}) {
  return (
    <ListRow
      title={session.device}
      meta={metaOf(session)}
      trailing={
        <Button
          size="small"
          aria-label={session.current ? "Sign out of this device" : `Sign out ${session.device}`}
          onClick={() => {
            if (session.current) {
              askSignOut();
            } else {
              end({ kind: "one", session });
            }
          }}
        >
          Sign out
        </Button>
      }
    />
  );
}

function EndDialogs({ ending, end }: { ending: Ending; end: (ending: Ending) => void }) {
  const endOne = useEndSession();
  const endOthers = useEndOtherSessions();
  const [shown, setShown] = useState(ending);
  if (ending !== null && ending !== shown) {
    setShown(ending);
  }
  const close = (open: boolean) => {
    if (!open) {
      end(null);
    }
  };
  return (
    <>
      <ConfirmDialog
        title={
          shown?.kind === "one" ? `Sign out ${shown.session.device}?` : "Sign out this device?"
        }
        confirmLabel="Sign out"
        open={ending?.kind === "one"}
        onOpenChange={close}
        onConfirm={() =>
          shown?.kind === "one" ? endOne.mutateAsync({ id: shown.session.id }) : undefined
        }
      >
        {shown?.kind === "one" && (
          <p>
            It last used SlipStream from {shown.session.address} at{" "}
            {clock(shown.session.lastSeenAt)}.
          </p>
        )}
      </ConfirmDialog>
      <ConfirmDialog
        title="Sign out every other device?"
        confirmLabel="Sign out others"
        open={ending?.kind === "others"}
        onOpenChange={close}
        onConfirm={() => endOthers.mutateAsync({ params: { others: true } })}
      >
        <p>This device stays signed in.</p>
      </ConfirmDialog>
    </>
  );
}

export function Devices() {
  const { sessions } = useListSessionsSuspense().data;
  const [ending, end] = useState<Ending>(null);
  const ordered = sessions.toSorted((a, b) => Number(b.current) - Number(a.current));
  return (
    <Section title="Devices" count={sessions.length}>
      <div className="flex flex-col items-start gap-3">
        <div className="self-stretch">
          <List label="Devices">
            {ordered.map((session) => (
              <DeviceRow key={session.id} session={session} end={end} />
            ))}
          </List>
        </div>
        {sessions.some((session) => !session.current) && (
          <Button
            onClick={() => {
              end({ kind: "others" });
            }}
          >
            Sign out every other device
          </Button>
        )}
      </div>
      <EndDialogs ending={ending} end={end} />
    </Section>
  );
}
