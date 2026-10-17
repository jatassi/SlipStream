import { Alert02Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { Button } from "./button";
import { Dialog } from "./dialog";
import { Icon } from "./icon";
import { useShell } from "./shell-context";

export type ConfirmDialogProps = {
  title: string;
  confirmLabel: string;
  look?: "primary" | "fail";
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<unknown> | undefined;
  children?: ReactNode;
};

function useConfirm({
  onConfirm,
  onOpenChange,
}: Pick<ConfirmDialogProps, "onConfirm" | "onOpenChange">) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const confirm = async () => {
    setBusy(true);
    setFailure(undefined);
    try {
      await onConfirm();
      onOpenChange(false);
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };
  const close = (open: boolean) => {
    setFailure(undefined);
    onOpenChange(open);
  };
  return { busy, failure, confirm, close };
}

/** A yes-or-no question on the kit's Dialog, a bottom sheet on phone. It closes once onConfirm settles. */
export function ConfirmDialog({
  title,
  confirmLabel,
  look = "primary",
  open,
  children,
  ...handlers
}: ConfirmDialogProps) {
  const { busy, failure, confirm, close } = useConfirm(handlers);
  const stacked = useShell() === "phone";
  return (
    <Dialog title={title} open={open} onOpenChange={close}>
      <div className="flex flex-col gap-4 text-ink-2">
        {children}
        {failure === undefined ? null : (
          <p role="alert" className="flex items-center gap-1.5 text-fail-ink">
            <Icon icon={Alert02Icon} className="size-4 shrink-0" />
            {failure}
          </p>
        )}
      </div>
      <div className={`mt-6 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
        <Button
          disabled={busy}
          onClick={() => {
            close(false);
          }}
        >
          Cancel
        </Button>
        <Button look={look} busy={busy} onClick={() => void confirm()}>
          {confirmLabel}
        </Button>
      </div>
    </Dialog>
  );
}
