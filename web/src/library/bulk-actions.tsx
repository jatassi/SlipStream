import {
  Delete02Icon,
  Notification01Icon,
  NotificationOff01Icon,
  RefreshIcon,
  Search01Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { type ReactNode, useState } from "react";

import { sentenceOf } from "../api/errors";
import { toast } from "../ui/toast";
import { useSearchSeveral } from "../wanted/several";
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
  /** A desktop bar keeps it in its More actions menu, so the bar stays one row. */
  readonly secondary: boolean;
  readonly onSelect: () => void;
};

const plain = {
  icon: undefined,
  busy: false,
  danger: false,
  oneTypeOnly: false,
  secondary: false,
} as const;
const more = { ...plain, secondary: true } as const;

/** Searches the picked Works in the bulk lane, asking first above 100; `dialog` holds the question. */
export function useBulkSearch(picked: readonly LibraryWork[]): {
  action: BulkAction;
  dialog: ReactNode;
} {
  const [first] = picked;
  const oneType = picked.every(({ type }) => type.id === first?.type.id);
  const several = useSearchSeveral(oneType ? first?.type : undefined);
  return {
    action: {
      ...plain,
      label: "Search",
      icon: Search01Icon,
      busy: several.busy,
      onSelect: () => {
        several.ask({ workIds: picked.map(({ id }) => id) }, picked.length);
      },
    },
    dialog: several.dialog,
  };
}

export function useBulkActions(
  picked: readonly LibraryWork[],
  onDialog: (kind: BulkDialogKind) => void,
  search: BulkAction,
): readonly BulkAction[] {
  const { running, on } = useRun(picked);
  const open = (kind: BulkDialogKind) => () => {
    onDialog(kind);
  };
  const manyItems = picked.length > 0 && picked.every(({ type }) => type.items === "many");
  return [
    search,
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
    ...(manyItems
      ? [{ ...plain, label: "Set monitoring…", oneTypeOnly: true, onSelect: open("monitoring") }]
      : []),
    { ...plain, label: "Change Profile…", oneTypeOnly: true, onSelect: open("profile") },
    { ...more, label: "Move to…", oneTypeOnly: true, onSelect: open("move") },
    { ...more, label: "Set library root…", oneTypeOnly: true, onSelect: open("root") },
    {
      ...more,
      label: "Refresh metadata",
      icon: RefreshIcon,
      busy: running === "refresh",
      onSelect: on({ kind: "refresh" }),
    },
    { ...more, label: "Delete…", icon: Delete02Icon, danger: true, onSelect: open("delete") },
  ];
}
