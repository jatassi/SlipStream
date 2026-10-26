import {
  Delete02Icon,
  Notification01Icon,
  NotificationOff01Icon,
  RefreshIcon,
} from "@hugeicons/core-free-icons";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { act } from "../works/actions/act";
import type { WorkAction } from "../works/actions/actions";
import type { BulkDialogKind } from "./bulk-dialogs";
import type { LibraryWork } from "./rows";

type Direct = WorkAction<"monitor" | "refresh">;

/** Runs an action that needs no dialog, toasting how it went per Work, or why it was refused. */
function useRun(picked: readonly LibraryWork[]) {
  const [running, setRunning] = useState<Direct["kind"]>();
  const run = async (action: Direct) => {
    setRunning(action.kind);
    try {
      await act(picked, action);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setRunning(undefined);
    }
  };
  return {
    running,
    on: (action: Direct) => () => {
      void run(action);
    },
  };
}

export function BulkActions({
  picked,
  oneType,
  onDialog,
}: {
  picked: readonly LibraryWork[];
  /** Profiles and roots belong to one Content Type, so only picks of one type change them. */
  oneType: boolean;
  onDialog: (kind: BulkDialogKind) => void;
}) {
  const { running, on } = useRun(picked);
  const open = (kind: BulkDialogKind) => () => {
    onDialog(kind);
  };
  return (
    <>
      <Button
        icon={Notification01Icon}
        busy={running === "monitor"}
        onClick={on({ kind: "monitor", on: true })}
      >
        Monitor
      </Button>
      <Button
        icon={NotificationOff01Icon}
        busy={running === "monitor"}
        onClick={on({ kind: "monitor", on: false })}
      >
        Unmonitor
      </Button>
      {oneType ? (
        <>
          <Button onClick={open("profile")}>Change Profile…</Button>
          <Button onClick={open("root")}>Set library root…</Button>
        </>
      ) : (
        <span className="text-meta text-ink-3">
          Pick Works of one Content Type to change their Profile or library root.
        </span>
      )}
      <Button icon={RefreshIcon} busy={running === "refresh"} onClick={on({ kind: "refresh" })}>
        Refresh metadata
      </Button>
      <Button icon={Delete02Icon} look="ghost" onClick={open("delete")}>
        Delete…
      </Button>
    </>
  );
}
