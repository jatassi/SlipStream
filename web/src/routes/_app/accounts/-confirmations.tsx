import type { IdentityAccountRow } from "../../../api/gen/model";
import {
  deleteAccount,
  endAccountSessions,
  updateAccount,
} from "../../../api/gen/query/identity/identity";
import type { ButtonLook } from "../../../ui/button";
import { ConfirmDialog } from "../../../ui/confirm-dialog";
import { EmailDialog } from "./-email-dialog";
import { ResetDialog } from "./-reset-dialog";

export type Confirmable = "role" | "access" | "sign-out-everywhere" | "delete";
export type Asking = Confirmable | "email" | "reset";

type Confirmation = {
  readonly title: string;
  readonly confirmLabel: string;
  readonly look: Extract<ButtonLook, "primary" | "fail">;
  readonly body: string;
  readonly run: () => Promise<unknown>;
};

type Confirmations = { readonly [A in Confirmable]: Confirmation };

const roleOf = ({ id, displayName: name, admin }: IdentityAccountRow): Confirmation =>
  admin
    ? {
        title: `Make ${name} a Member?`,
        confirmLabel: "Make Member",
        look: "fail",
        body: "They lose the Admin pages at once, in every tab they have open.",
        run: () => updateAccount(id, { admin: false }),
      }
    : {
        title: `Make ${name} an Admin?`,
        confirmLabel: "Make Admin",
        look: "primary",
        body: "They can run SlipStream, including Accounts and Settings.",
        run: () => updateAccount(id, { admin: true }),
      };

const accessOf = ({ id, displayName: name, disabled }: IdentityAccountRow): Confirmation =>
  disabled
    ? {
        title: `Turn on ${name}?`,
        confirmLabel: "Turn on",
        look: "primary",
        body: "They can sign in again.",
        run: () => updateAccount(id, { disabled: false }),
      }
    : {
        title: `Turn off ${name}?`,
        confirmLabel: "Turn off",
        look: "fail",
        body: "They're signed out everywhere and can't sign in until an Admin turns them on.",
        run: () => updateAccount(id, { disabled: true }),
      };

function confirmationsOf(
  account: IdentityAccountRow,
  onDeleted: () => Promise<void>,
): Confirmations {
  const { id, displayName: name } = account;
  return {
    role: roleOf(account),
    access: accessOf(account),
    "sign-out-everywhere": {
      title: `Sign ${name} out everywhere?`,
      confirmLabel: "Sign out everywhere",
      look: "fail",
      body: "Every device they're signed in on goes back to Sign in.",
      run: () => endAccountSessions(id),
    },
    delete: {
      title: `Delete ${name}?`,
      confirmLabel: "Delete",
      look: "fail",
      body: "Their Account, passkeys and sessions are gone for good.",
      run: async () => {
        await deleteAccount(id);
        await onDeleted();
      },
    },
  };
}

function ConfirmDialogs({
  confirms,
  asking,
  onClose,
}: {
  confirms: Confirmations;
  asking: Asking | undefined;
  onClose: () => void;
}) {
  return Object.entries(confirms).map(([id, { body, run, ...confirm }]) => (
    <ConfirmDialog
      key={id}
      {...confirm}
      open={asking === id}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onConfirm={run}
    >
      <p>{body}</p>
    </ConfirmDialog>
  ));
}

export function AccountDialogs({
  account,
  asking,
  onClose,
  onDeleted,
}: {
  account: IdentityAccountRow;
  asking: Asking | undefined;
  onClose: () => void;
  onDeleted: () => Promise<void>;
}) {
  return (
    <>
      <ConfirmDialogs
        confirms={confirmationsOf(account, onDeleted)}
        asking={asking}
        onClose={onClose}
      />
      <EmailDialog account={account} open={asking === "email"} onClose={onClose} />
      <ResetDialog account={account} open={asking === "reset"} onClose={onClose} />
    </>
  );
}
