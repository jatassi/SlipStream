import { createPortal } from "react-dom";

import { useShellSlots } from "../shell/page";
import type { ReachabilityHopView } from "../api/gen/model";
import { Inspector } from "../ui/inspector";
import { Sheet } from "../ui/sheet";
import { HopChecks } from "./hop-checks";
import { hopNames } from "./words";

export type SelectedHop = {
  readonly subject: string;
  readonly hop: ReachabilityHopView;
  readonly judgedAt: string | null;
};

/**
 * A selected hop's checks, each leading with what it means and its fix, with
 * the values under Details: in the desktop shell's Inspector column, and in a
 * sheet on phone.
 */
export function HopInspector({
  selected,
  onClose,
}: {
  selected: SelectedHop | undefined;
  onClose: () => void;
}) {
  const slot = useShellSlots().inspector;
  const title =
    selected === undefined ? "" : `${hopNames[selected.hop.kind]} · ${selected.subject}`;
  if (slot === null) {
    return (
      <Sheet
        title={title}
        open={selected !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            onClose();
          }
        }}
      >
        <div className="flex flex-col gap-3">
          {selected !== undefined && <HopChecks hop={selected.hop} judgedAt={selected.judgedAt} />}
        </div>
      </Sheet>
    );
  }
  return selected === undefined
    ? null
    : createPortal(
        <Inspector title={title} onClose={onClose}>
          <HopChecks hop={selected.hop} judgedAt={selected.judgedAt} />
        </Inspector>,
        slot,
      );
}
