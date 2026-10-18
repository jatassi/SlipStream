import { createPortal } from "react-dom";

import type { Part } from "../../../../diagram/parts-order";
import { useShellSlots } from "../../../../shell/page";
import { Inspector } from "../../../../ui/inspector";
import { Sheet } from "../../../../ui/sheet";
import type { Acting } from "../-manage/dialogs";
import { wordsOf } from "./inspect-words";
import type { Landing } from "./view";

/**
 * What a selected part of the Diagram is, in plain words with the machinery
 * under Details: in the desktop shell's Inspector column, and in a sheet on
 * phone.
 */
export function DiagramInspector({
  landing,
  part,
  onClose,
  act,
}: {
  landing: Landing;
  part: Part | undefined;
  onClose: () => void;
  act: (acting: Acting) => void;
}) {
  const slot = useShellSlots().inspector;
  const words = part === undefined ? undefined : wordsOf(landing, part, act);
  if (slot === null) {
    return (
      <Sheet
        title={words?.title ?? ""}
        open={words !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            onClose();
          }
        }}
      >
        <div className="flex flex-col gap-3">{words?.body}</div>
      </Sheet>
    );
  }
  return words === undefined
    ? null
    : createPortal(
        <Inspector title={words.title} onClose={onClose}>
          {words.body}
        </Inspector>,
        slot,
      );
}
