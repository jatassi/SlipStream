import { Link } from "@tanstack/react-router";
import { useState } from "react";

import { IssuedLink } from "../../../accounts/issued-link";
import { isProblem } from "../../../api/errors";
import type { IdentityAccountRow, IdentityIssuedResetLink } from "../../../api/gen/model";
import { useCreateResetLink } from "../../../api/gen/query/identity/identity";
import { Button } from "../../../ui/button";
import { Checkbox } from "../../../ui/checkbox";
import { Dialog, DialogClose } from "../../../ui/dialog";

function Issued({
  account,
  issued,
}: {
  account: IdentityAccountRow;
  issued: IdentityIssuedResetLink;
}) {
  return (
    <div className="flex flex-col gap-6">
      <IssuedLink label="Reset link" url={issued.url} to={account.email} mail={issued.email} />
      <p className="text-ink-2">SlipStream shows this link only once.</p>
      <div className="flex justify-end">
        <DialogClose>
          <Button look="primary">Done</Button>
        </DialogClose>
      </div>
    </div>
  );
}

function Refusal({ message, addressUnset }: { message: string; addressUnset: boolean }) {
  return (
    <div role="alert" className="flex flex-col items-start gap-2 text-fail-ink">
      <p>{message}</p>
      {addressUnset ? (
        <Button
          nativeButton={false}
          render={<Link to="/settings/$section" params={{ section: "address" }} />}
        >
          Set the address
        </Button>
      ) : null}
    </div>
  );
}

function ResetBody({ account }: { account: IdentityAccountRow }) {
  const [removePasskeys, setRemovePasskeys] = useState(false);
  const reset = useCreateResetLink();
  if (reset.data !== undefined) {
    return <Issued account={account} issued={reset.data} />;
  }
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">
        The link lets {account.displayName} set a new PIN. Sending it signs them out everywhere.
      </p>
      <div className="flex flex-col gap-1">
        <Checkbox
          label="Also remove passkeys"
          checked={removePasskeys}
          onCheckedChange={setRemovePasskeys}
        />
        <p className="text-meta text-ink-3">Turn this on if someone else may have had access.</p>
      </div>
      {reset.error === null ? null : (
        <Refusal
          message={reset.error.message}
          addressUnset={isProblem(reset.error, "address_unset")}
        />
      )}
      <div className="mt-2 flex justify-end gap-2">
        <DialogClose>
          <Button disabled={reset.isPending}>Cancel</Button>
        </DialogClose>
        <Button
          look="primary"
          busy={reset.isPending}
          onClick={() => {
            reset.mutate({ id: account.id, data: { removePasskeys } });
          }}
        >
          Send the Reset link
        </Button>
      </div>
    </div>
  );
}

export function ResetDialog({
  account,
  open,
  onClose,
}: {
  account: IdentityAccountRow;
  open: boolean;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={`Send ${account.displayName} a Reset link`}
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <ResetBody account={account} />
    </Dialog>
  );
}
