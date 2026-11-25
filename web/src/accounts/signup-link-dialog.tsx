import { Link01Icon } from "@hugeicons/core-free-icons";
import { Suspense, useState } from "react";

import type { IdentityIssuedSignupLink } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { Spinner } from "../ui/spinner";
import { IssuedLink } from "./issued-link";
import { LinkForm } from "./signup-link-form";

const list = new Intl.ListFormat("en", { type: "conjunction" });

function droppedSentence(dropped: readonly string[]): string {
  const names = list.format(dropped);
  return dropped.length === 1
    ? `${names} is no longer shown to Members and isn't offered by the new link.`
    : `${names} are no longer shown to Members and aren't offered by the new link.`;
}

export function IssuedSignupLink({ issued }: { issued: IdentityIssuedSignupLink }) {
  const dropped = issued.dropped ?? [];
  return (
    <div className="flex flex-col gap-6">
      <IssuedLink label="Signup link" url={issued.url} to={issued.link.email} mail={issued.email} />
      {dropped.length > 0 && <p>{droppedSentence(dropped)}</p>}
      <div className="flex justify-end">
        <DialogClose>
          <Button look="primary">Done</Button>
        </DialogClose>
      </div>
    </div>
  );
}

function SignupLinkBody() {
  const [issued, setIssued] = useState<IdentityIssuedSignupLink>();
  return issued === undefined ? (
    <Suspense fallback={<Spinner />}>
      <LinkForm onIssued={setIssued} />
    </Suspense>
  ) : (
    <IssuedSignupLink issued={issued} />
  );
}

export function NewSignupLinkDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="New Signup link" open={open} onOpenChange={onOpenChange}>
      <SignupLinkBody />
    </Dialog>
  );
}

export function NewSignupLinkButton({ onClick }: { onClick: () => void }) {
  return (
    <Button look="primary" icon={Link01Icon} onClick={onClick}>
      New Signup link
    </Button>
  );
}
