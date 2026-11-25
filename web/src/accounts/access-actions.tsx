import { type ReactElement, Suspense, useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  MailOutcome,
  MediaserversAccessView as Access,
  MediaserversActed as Acted,
  MediaserversAccessViewActionsItem as Action,
  MediaserversAccessViewState as State,
  MediaserversServerUser as ServerUser,
} from "../api/gen/model";
import {
  linkMediaServerUser,
  offerMediaServer,
  removeMediaServerAccess,
  useListMediaServerUsersSuspense,
} from "../api/gen/query/mediaservers/mediaservers";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Dialog, DialogClose } from "../ui/dialog";
import { Empty } from "../ui/empty";
import { submitErrors, useAppForm } from "../ui/form";
import { RadioGroup } from "../ui/radio";
// oxlint-disable-next-line import/max-dependencies -- each action draws its own button and dialog
import { Skeleton } from "../ui/skeleton";
import { toast } from "../ui/toast";
import { AddToPlexHome, CheckAgain, ShareByHand, ShareOnPlex } from "./plex-share";

const accountName = (row: Access) => row.plex?.username ?? row.user?.username;

const waitingFor: {
  readonly [W in Access["server"]["access"]]: (row: Access) => string;
} = {
  plex_tv: ({ member }) => `Waiting for ${member.name} to sign in with Plex.`,
  users: ({ member }) => `Waiting for ${member.name} to make an account.`,
  users_linked: ({ server }) => `Waiting for you to link an account on ${server.name}.`,
};

const lines: { readonly [S in State]: (row: Access) => string } = {
  offered: (row) => waitingFor[row.server.access](row),
  waiting: ({ plex }) =>
    `Signed in through Plex as ${[plex?.username, plex?.email].filter(Boolean).join(", ")}.`,
  shared: ({ member }) => `Waiting for ${member.name} to accept the share in Plex's email.`,
  making: ({ member }) => `Making ${member.name}'s account…`,
  active: (row) => {
    const name = accountName(row);
    return name === undefined ? "Can watch." : `Can watch as ${name}.`;
  },
  removing: ({ member, server, problem }) =>
    [`SlipStream couldn't remove ${member.name}'s access to ${server.name}.`, problem?.detail]
      .filter(Boolean)
      .join(" "),
  removed: ({ server }) => `Access removed. The account on ${server.name} is turned off.`,
};

/** What a row's access is now, for an Admin, with any failure's detail. */
export const accessLine = (row: Access): string => lines[row.state](row);

const removeWords = ({ member, server, state, plex }: Access): string => {
  if (state !== "shared" && state !== "active") {
    return `${server.name} is no longer offered to ${member.name}.`;
  }
  if (plex?.plexHome === true) {
    return `This removes ${member.name} from your Plex Home, which ends their access to every server you own.`;
  }
  return plex === undefined
    ? `SlipStream turns off ${member.name}'s account on ${server.name}. Offering ${server.name} again turns it back on.`
    : `${member.name} can no longer watch ${server.name} in Plex. Other servers you share with them stay shared.`;
};

function RemoveAccess({ row }: { row: Access }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Remove access
      </Button>
      <ConfirmDialog
        title={`Remove ${row.member.name}'s access to ${row.server.name}?`}
        confirmLabel="Remove access"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          await removeMediaServerAccess(row.id);
          toast(`Removed ${row.member.name}'s access to ${row.server.name}`);
        }}
      >
        <p>{removeWords(row)}</p>
      </ConfirmDialog>
    </>
  );
}

const mailWords: {
  readonly [O in MailOutcome["outcome"]]: (row: Access, detail: string | undefined) => string;
} = {
  sent: ({ member, server }) => `Emailed ${member.name} about ${server.name}`,
  failed: ({ member, server }, detail) =>
    `The email to ${member.name} failed${detail === undefined ? "" : `: ${detail}`}. Tell ${member.name} about ${server.name} yourself.`,
  "not-set-up": ({ member, server }) =>
    `Email isn't set up. Tell ${member.name} about ${server.name} yourself.`,
};

/** Toasts what an operation did, then what became of the email it sent the Member. */
export function toastActed({ row, mail }: Acted, words: string) {
  toast(words);
  if (row !== undefined && mail !== undefined) {
    toast(mailWords[mail.outcome](row, mail.detail), {
      kind: mail.outcome === "sent" ? "done" : "fail",
    });
  }
}

/**
 * Runs one access operation from a button and toasts what it did, or why it
 * was refused. The live channel's media-servers.access-changed reads the access again.
 */
export function useAccessRun(
  act: () => Promise<Acted>,
  words: string | ((acted: Acted) => string),
) {
  const [busy, setBusy] = useState(false);
  const run = () => {
    setBusy(true);
    void act()
      .then((acted) => {
        toastActed(acted, typeof words === "string" ? words : words(acted));
        return null;
      })
      .catch((error: unknown) => {
        toast(sentenceOf(error), { kind: "fail" });
      })
      .finally(() => {
        setBusy(false);
      });
  };
  return { busy, run };
}

function TryAgain({ row }: { row: Access }) {
  const { busy, run } = useAccessRun(
    () => removeMediaServerAccess(row.id),
    `Removed ${row.member.name}'s access to ${row.server.name}`,
  );
  return (
    <Button size="small" busy={busy} onClick={run}>
      Try again
    </Button>
  );
}

function OfferAgain({ row, account }: { row: Access; account: string }) {
  const { busy, run } = useAccessRun(
    () => offerMediaServer(account, row.server.id, {}),
    `Offered ${row.server.name} to ${row.member.name} again`,
  );
  return (
    <Button size="small" busy={busy} onClick={run}>
      Offer again
    </Button>
  );
}

function useLinkForm(row: Access, first: ServerUser, onDone: () => void) {
  return useAppForm({
    defaultValues: { userId: first.id },
    onSubmit: async ({ value, formApi }) => {
      const linked = await linkMediaServerUser(row.id, value).catch((error: unknown) => {
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
        return null;
      });
      if (linked !== null) {
        toast(`Linked ${row.member.name} to an account on ${row.server.name}`);
        onDone();
      }
    },
  });
}

function LinkForm({
  row,
  users,
  onDone,
}: {
  row: Access;
  users: readonly [ServerUser, ...ServerUser[]];
  onDone: () => void;
}) {
  const form = useLinkForm(row, users[0], onDone);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p className="text-ink-2">
        {`${row.member.name} signs in to ${row.server.name} as this user and keeps its libraries.`}
      </p>
      <form.AppForm>
        <form.Field name="userId">
          {(field) => (
            <RadioGroup
              label={`Users on ${row.server.name}`}
              options={users.map((user) => ({ value: user.id, label: user.name }))}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
        <form.FormAlert />
        <div className="flex justify-end gap-2">
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Link" />
        </div>
      </form.AppForm>
    </form>
  );
}

function LinkUsers({ row, onDone }: { row: Access; onDone: () => void }) {
  const { users } = useListMediaServerUsersSuspense(row.server.id).data;
  const [first, ...rest] = users;
  return first === undefined ? (
    <Empty look="inline">
      {`${row.server.name} has no account to link. Administrators, turned-off users and users linked to other Members aren't listed.`}
    </Empty>
  ) : (
    <LinkForm row={row} users={[first, ...rest]} onDone={onDone} />
  );
}

function LinkAccount({ row }: { row: Access }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Link an existing account
      </Button>
      <Dialog
        title={`Link ${row.member.name} to an account on ${row.server.name}`}
        open={open}
        onOpenChange={setOpen}
      >
        <Suspense fallback={<Skeleton label="Loading the users" shape="rows" lines={1} />}>
          <LinkUsers
            row={row}
            onDone={() => {
              setOpen(false);
            }}
          />
        </Suspense>
      </Dialog>
    </>
  );
}

type ActionButton = (props: { row: Access }) => ReactElement | null;

/** The button each action draws. */
const buttons: { readonly [A in Action]: ActionButton } = {
  shareFriend: ShareOnPlex,
  shareHome: AddToPlexHome,
  markAdded: ShareByHand,
  check: CheckAgain,
  link: LinkAccount,
  offer: ({ row }) =>
    row.member.accountId === undefined ? null : (
      <OfferAgain row={row} account={row.member.accountId} />
    ),
  retryRemove: TryAgain,
  remove: RemoveAccess,
};

/** The buttons for what an Admin may start on a row, exactly as its actions name them. */
export function AccessActions({ row }: { row: Access }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {row.actions.map((action) => {
        const Draw = buttons[action];
        return <Draw key={action} row={row} />;
      })}
    </div>
  );
}
