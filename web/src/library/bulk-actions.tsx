import {
  Delete02Icon,
  Notification01Icon,
  NotificationOff01Icon,
  RefreshIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
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

export type BulkAction = {
  readonly label: string;
  readonly icon: IconSvgElement | undefined;
  readonly busy: boolean;
  readonly danger: boolean;
  /** Profiles and roots belong to one Content Type, so only picks of one type change them. */
  readonly oneTypeOnly: boolean;
  readonly onSelect: () => void;
};

const plain = { icon: undefined, busy: false, danger: false, oneTypeOnly: false } as const;

export function useBulkActions(
  picked: readonly LibraryWork[],
  onDialog: (kind: BulkDialogKind) => void,
): readonly BulkAction[] {
  const { running, on } = useRun(picked);
  const open = (kind: BulkDialogKind) => () => {
    onDialog(kind);
  };
  return [
    {
      ...plain,
      label: "Monitor",
      icon: Notification01Icon,
      busy: running === "monitor",
      onSelect: on({ kind: "monitor", on: true }),
    },
    {
      ...plain,
      label: "Unmonitor",
      icon: NotificationOff01Icon,
      busy: running === "monitor",
      onSelect: on({ kind: "monitor", on: false }),
    },
    { ...plain, label: "Change Profile…", oneTypeOnly: true, onSelect: open("profile") },
    { ...plain, label: "Set library root…", oneTypeOnly: true, onSelect: open("root") },
    {
      ...plain,
      label: "Refresh metadata",
      icon: RefreshIcon,
      busy: running === "refresh",
      onSelect: on({ kind: "refresh" }),
    },
    { ...plain, label: "Delete…", icon: Delete02Icon, danger: true, onSelect: open("delete") },
  ];
}
