import { useQuery } from "@tanstack/react-query";

import type { IdentityIssuedSignupLink, IdentityPendingLink } from "../api/gen/model";
import {
  deleteSignupLink,
  getGetSignupLinkQueryKey,
  getSignupLink,
  resendSignupLink,
} from "../api/gen/query/identity/identity";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Dialog, DialogClose } from "../ui/dialog";
import { ReadOnlyField } from "../ui/read-only-field";
import { Spinner } from "../ui/spinner";
import { IssuedSignupLink } from "./signup-link-dialog";

export type Acting =
  | { readonly kind: "copy"; readonly link: IdentityPendingLink }
  | { readonly kind: "resend"; readonly link: IdentityPendingLink }
  | { readonly kind: "resent"; readonly issued: IdentityIssuedSignupLink }
  | { readonly kind: "withdraw"; readonly link: IdentityPendingLink };

export type SetActing = (update: (acting: Acting | undefined) => Acting | undefined) => void;

type Props = { acting: Acting | undefined; setActing: SetActing };

// A dialog closes only its own kind, so a confirm that hands over to the next
// dialog doesn't close that one.
const closer = (setActing: SetActing, kind: Acting["kind"]) => (open: boolean) => {
  if (!open) {
    setActing((current) => (current?.kind === kind ? undefined : current));
  }
};

function CopiedUrl({ link }: { link: IdentityPendingLink }) {
  const query = useQuery({
    queryKey: getGetSignupLinkQueryKey(link.id),
    queryFn: ({ signal }) => getSignupLink(link.id, { signal }),
  });
  if (query.data !== undefined) {
    return (
      <ReadOnlyField
        label="Signup link"
        value={query.data.url}
        description="Anyone with this link can use it, so send it only to them."
      />
    );
  }
  return query.error === null ? (
    <Spinner />
  ) : (
    <p role="alert" className="text-fail-ink">
      {query.error.message}
    </p>
  );
}

function CopyDialog({ acting, setActing }: Props) {
  return (
    <Dialog
      title={acting?.kind === "copy" ? `Signup link for ${acting.link.email}` : "Signup link"}
      open={acting?.kind === "copy"}
      onOpenChange={closer(setActing, "copy")}
    >
      {acting?.kind === "copy" ? (
        <div className="flex flex-col gap-6">
          <CopiedUrl link={acting.link} />
          <div className="flex justify-end">
            <DialogClose>
              <Button look="primary">Done</Button>
            </DialogClose>
          </div>
        </div>
      ) : null}
    </Dialog>
  );
}

function ResendDialogs({ acting, setActing }: Props) {
  return (
    <>
      <ConfirmDialog
        title="Send a new Signup link?"
        confirmLabel="Send again"
        open={acting?.kind === "resend"}
        onOpenChange={closer(setActing, "resend")}
        onConfirm={async () => {
          if (acting?.kind === "resend") {
            const issued = await resendSignupLink(acting.link.id);
            setActing(() => ({ kind: "resent", issued }));
          }
        }}
      >
        <p>
          {acting?.kind === "resend" ? acting.link.email : "They"} get a new link, and the old one
          stops working.
        </p>
      </ConfirmDialog>
      <Dialog
        title="New Signup link"
        open={acting?.kind === "resent"}
        onOpenChange={closer(setActing, "resent")}
      >
        {acting?.kind === "resent" ? <IssuedSignupLink issued={acting.issued} /> : null}
      </Dialog>
    </>
  );
}

function WithdrawDialog({ acting, setActing }: Props) {
  return (
    <ConfirmDialog
      title="Withdraw the Signup link?"
      confirmLabel="Withdraw"
      look="fail"
      open={acting?.kind === "withdraw"}
      onOpenChange={closer(setActing, "withdraw")}
      onConfirm={() => (acting?.kind === "withdraw" ? deleteSignupLink(acting.link.id) : undefined)}
    >
      <p>The link for {acting?.kind === "withdraw" ? acting.link.email : "them"} stops working.</p>
    </ConfirmDialog>
  );
}

export function PendingLinkDialogs(props: Props) {
  return (
    <>
      <CopyDialog {...props} />
      <ResendDialogs {...props} />
      <WithdrawDialog {...props} />
    </>
  );
}
