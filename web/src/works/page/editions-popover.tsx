import type { ReactElement } from "react";

import { Dialog } from "../../ui/dialog";
import { Popover } from "../../ui/popover";
import { useShell } from "../../ui/shell-context";
import { type EditorProps, EditionsEditor } from "./editions-editor";

/**
 * The Work's Edition list, from its Profile or set for this Work, edited
 * through the Impact: a popover on its trigger in the desktop shell, and a
 * sheet on phone or with no trigger, as the phone hero's actions open it.
 */
export function EditionsPopover({
  trigger,
  open,
  onOpenChange,
  ...editor
}: Omit<EditorProps, "onDone"> & {
  readonly trigger: ReactElement | undefined;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}) {
  const close = () => {
    onOpenChange(false);
  };
  const body = <EditionsEditor {...editor} onDone={close} />;
  if (useShell() === "phone" || trigger === undefined) {
    return (
      <Dialog
        title={`Editions of ${editor.work.label}`}
        size={680}
        {...(trigger !== undefined && { trigger })}
        open={open}
        onOpenChange={onOpenChange}
      >
        {body}
      </Dialog>
    );
  }
  return (
    <Popover
      trigger={trigger}
      label={`Editions of ${editor.work.label}`}
      open={open}
      onOpenChange={onOpenChange}
    >
      {body}
    </Popover>
  );
}
