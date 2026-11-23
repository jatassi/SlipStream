import {
  Cancel01Icon,
  Delete02Icon,
  ArrowDataTransferHorizontalIcon,
  Edit02Icon,
  FileRemoveIcon,
  MoreHorizontalIcon,
  RefreshIcon,
  StarIcon,
} from "@hugeicons/core-free-icons";

import { sentenceOf } from "../api/errors";
import type { RootsRoot } from "../api/gen/model";
import { makeDefaultRoot, scanRoot } from "../api/gen/query/roots/roots";
import { cancelTaskRun } from "../api/gen/query/tasks/tasks";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuItem } from "../ui/menu";
import { toast } from "../ui/toast";
import type { Acting } from "./acting-dialogs";

async function calling(call: () => Promise<unknown>, done: string) {
  try {
    await call();
    toast(done);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

/** Cancel library scan while the root's unit runs, Mark them gone and Scan now after a scan stopped, else Scan now. */
function scanItems(root: RootsRoot, name: string, onAct: (acting: Acting) => void): MenuItem[] {
  const { running, stopped } = root.scan;
  if (running !== null) {
    return [
      {
        label: "Cancel library scan",
        icon: Cancel01Icon,
        onSelect: () =>
          void calling(
            () => cancelTaskRun(running.runId),
            "Library scan canceled for every root in the run. The files it already recorded stay recorded.",
          ),
      },
    ];
  }
  const scanNow: MenuItem = {
    label: "Scan now",
    icon: RefreshIcon,
    onSelect: () => void calling(() => scanRoot(root.id), `Library scan started for the ${name}.`),
  };
  return stopped === null
    ? [scanNow]
    : [
        {
          label: "Mark them gone",
          icon: FileRemoveIcon,
          onSelect: () => {
            onAct({ kind: "gone", rootId: root.id, name });
          },
        },
        scanNow,
      ];
}

/** Make default, unless the root is its type's default. */
const defaultItems = (root: RootsRoot, label: string): MenuItem[] =>
  root.default
    ? []
    : [
        {
          label: "Make default",
          icon: StarIcon,
          onSelect: () =>
            void calling(() => makeDefaultRoot(root.id), defaultWords(label, root.path)),
        },
      ];

/** Move root… while the root holds Works. */
const moveItems = (root: RootsRoot, onAct: (acting: Acting) => void): MenuItem[] =>
  root.works > 0
    ? [
        {
          label: "Move root…",
          icon: ArrowDataTransferHorizontalIcon,
          onSelect: () => {
            onAct({ kind: "move", rootId: root.id });
          },
        },
      ]
    : [];

/** The root's scan actions, Make default, Edit and Delete; a root on a written-off Node leads with Re-home. */
const defaultWords = (label: string, path: string) =>
  `New ${label.toLowerCase()} now go into ${path}.`;

export function RootMenu({
  root,
  label,
  name,
  onAct,
}: {
  root: RootsRoot;
  label: string;
  name: string;
  onAct: (acting: Acting) => void;
}) {
  const lost = root.state === "written_off";
  const edit: MenuItem = {
    label: lost ? "Re-home" : "Edit",
    icon: Edit02Icon,
    onSelect: () => {
      onAct({ kind: "edit", rootId: root.id });
    },
  };
  const makeDefault = defaultItems(root, label);
  return (
    <Menu
      label={root.path}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${root.path}`} />}
      items={[
        ...(lost
          ? [edit, ...makeDefault, "separator" as const, ...scanItems(root, name, onAct)]
          : [
              ...scanItems(root, name, onAct),
              "separator" as const,
              ...makeDefault,
              edit,
              ...moveItems(root, onAct),
            ]),
        "separator",
        {
          label: "Delete",
          icon: Delete02Icon,
          tone: "danger",
          onSelect: () => {
            onAct({ kind: "delete", rootId: root.id });
          },
        },
      ]}
    />
  );
}
