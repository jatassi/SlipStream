import { sentenceOf } from "../../api/errors";
import type {
  MediaserversMyAccess as MyAccess,
  MediaserversMyAccessRow as MyRow,
} from "../../api/gen/model";
import {
  useCheckMyMediaServerAccess,
  useUseAnotherPlexAccount,
} from "../../api/gen/query/mediaservers/mediaservers";
import { Button } from "../../ui/button";
import { ReadOnlyField } from "../../ui/read-only-field";
import { SignalLine } from "../../ui/signal-line";
import { Spinner } from "../../ui/spinner";
import { inAppBrowser } from "./in-app";
import type { Lapse, MemberPlexPin } from "./use-member-plex-pin";

export type BlockProps = {
  readonly row: MyRow;
  readonly mine: MyAccess;
  readonly pin: MemberPlexPin;
};

function plexName(mine: MyAccess): string {
  if (mine.plex === undefined) {
    throw new Error(
      "a Member's Plex row past offered comes with the Plex account they signed in as",
    );
  }
  return mine.plex.username;
}

const lapseWords: { readonly [L in Lapse]: (row: MyRow, mine: MyAccess) => string } = {
  expired: () => "The Plex sign-in timed out. Continue with Plex to start again.",
  blocked: () =>
    "Your browser blocked Plex's sign-in tab. Continue with Plex opens it in this tab instead.",
  plex_account_taken: () =>
    "That Plex account is linked to another Account. Use another Plex account.",
  plex_account_owner: (row) =>
    `That's the Plex account that owns ${row.server.name}. Sign in with your own Plex account.`,
  plex_account_other: (row, mine) => {
    const held =
      mine.rows.find(
        (other) =>
          other.server.access === "plex_tv" &&
          (other.state === "shared" || other.state === "active"),
      ) ?? row;
    const username = plexName(mine);
    return `${held.server.name} is shared with your Plex account ${username}. Sign in to Plex as ${username}.`;
  },
};

const inApp = () => inAppBrowser(navigator.userAgent) !== undefined;

function PlexSignIn({ label, row, mine, pin }: BlockProps & { label: string }) {
  if (inApp()) {
    return (
      <>
        <SignalLine tone="warn">
          Plex&apos;s sign-in doesn&apos;t work inside this app. Open this page in Safari or Chrome.
        </SignalLine>
        <ReadOnlyField label="Link" value={location.href} />
      </>
    );
  }
  if (pin.pin !== null) {
    return (
      <div className="flex flex-col gap-2">
        <p className="flex items-center gap-2 text-ink-1">
          <Spinner />
          Waiting for you to sign in to Plex.
        </p>
        <a
          href={pin.pin.authUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="self-start font-medium text-ink-2 underline hover:text-ink-1"
        >
          {pin.opened ? "Open Plex's sign-in again" : "Open Plex's sign-in"}
        </a>
      </div>
    );
  }
  return (
    <>
      {pin.lapse !== undefined && (
        <SignalLine tone="warn" announce>
          {lapseWords[pin.lapse](row, mine)}
        </SignalLine>
      )}
      {pin.failure !== undefined && (
        <SignalLine tone="fail" announce>
          {pin.failure}
        </SignalLine>
      )}
      <Button look="primary" busy={pin.busy} onClick={pin.start}>
        {label}
      </Button>
    </>
  );
}

export function PlexOffered(props: BlockProps) {
  const { row } = props;
  return (
    <>
      <p>
        {row.adminName}&apos;s movies and series play in Plex. Sign in to Plex so {row.adminName}{" "}
        can share {row.server.name} with you.
      </p>
      <PlexSignIn label="Continue with Plex" {...props} />
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        New to Plex? You can make a free account on Plex&apos;s page.
      </p>
    </>
  );
}

export function PlexWaiting({ row, mine }: BlockProps) {
  const another = useUseAnotherPlexAccount();
  const username = plexName(mine);
  return (
    <>
      {mine.mailSetUp ? (
        <p>
          You&apos;re signed in to Plex as {username}. {row.adminName} shares {row.server.name} with
          you next, and SlipStream emails you when you can watch. You may also get an email from
          Plex, and you don&apos;t need to do anything with it.
        </p>
      ) : (
        <p>
          You&apos;re signed in to Plex as {username}. SlipStream can&apos;t email {row.adminName}.
          Tell them you signed in to Plex.
        </p>
      )}
      {another.error !== null && (
        <SignalLine tone="fail" announce>
          {sentenceOf(another.error)}
        </SignalLine>
      )}
      {row.actions.includes("anotherPlex") && (
        <Button
          look="ghost"
          busy={another.isPending}
          onClick={() => {
            another.mutate();
          }}
        >
          Use another Plex account
        </Button>
      )}
    </>
  );
}

function CheckAgain({ row }: { row: MyRow }) {
  const check = useCheckMyMediaServerAccess();
  return (
    <>
      {check.data?.state === "shared" && (
        <SignalLine tone="warn" announce>
          The share isn&apos;t accepted yet. Accept it in the email from Plex.
        </SignalLine>
      )}
      {check.error !== null && (
        <SignalLine tone="fail" announce>
          {sentenceOf(check.error)}
        </SignalLine>
      )}
      <Button
        busy={check.isPending}
        onClick={() => {
          check.mutate({ accessId: row.id });
        }}
      >
        Check again
      </Button>
    </>
  );
}

export function PlexShared(props: BlockProps) {
  const { row } = props;
  return (
    <>
      <p>
        {row.adminName} shared {row.server.name} with you. Open the email from Plex and accept the
        share, or sign in with Plex again and SlipStream accepts it for you.
      </p>
      <PlexSignIn label="Sign in with Plex again" {...props} />
      {row.actions.includes("check") && <CheckAgain row={row} />}
    </>
  );
}
