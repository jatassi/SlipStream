import {
  ArrowUpRight01Icon,
  Cancel01Icon,
  FileImportIcon,
  InformationCircleIcon,
  MoreHorizontalIcon,
} from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { ActionsMenu } from "../torrents/actions-menu";
import { type TorrentAction, menuFor, subjectOf } from "../torrents/actions";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { type GrabView, GrabDialogs } from "./grab-details";
import type { Job } from "./job";
import { byHand } from "./stages";

/** A Grab's own menu items around its Torrent's: Details first, Cancel Grab last. */
type GrabItems = (torrentItems: readonly MenuEntry[]) => MenuEntry[];

const jobActions = new Set<TorrentAction>(["pause", "resume", "remove"]);

function TorrentMenu({
  grab,
  work,
  row,
  names,
  itemsWith,
}: Omit<Job, "torrent"> & { row: NonNullable<Job["torrent"]>; itemsWith: GrabItems }) {
  const navigate = useNavigate();
  const actions =
    grab.stage === "downloading" ? menuFor(row).filter((action) => jobActions.has(action)) : [];
  return (
    <ActionsMenu
      subject={subjectOf(row, work)}
      actions={actions}
      names={names}
      extra={itemsWith([
        {
          label: "Open its torrent",
          icon: ArrowUpRight01Icon,
          onSelect: () => {
            void navigate({
              to: "/activity/torrents/$torrentId",
              params: { torrentId: row.id },
            });
          },
        },
      ])}
    />
  );
}

function GrabMenu({ work, items }: { work: string; items: readonly MenuEntry[] }) {
  const label = `${work} actions`;
  return (
    <Menu
      label={label}
      trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
      items={items}
    />
  );
}

/**
 * A Grab's menu: its Details, Cancel while it can be canceled, and once it
 * has a Torrent, opening it and, while it downloads, pause, resume or remove.
 */
export function JobMenu(job: Job) {
  const { grab, work, torrent } = job;
  const [view, setView] = useState<GrabView>();
  const itemsWith: GrabItems = (torrentItems) => [
    {
      label: "Details",
      icon: InformationCircleIcon,
      onSelect: () => {
        setView("details");
      },
    },
    ...torrentItems,
    ...(byHand(grab)
      ? [
          {
            label: "Import by hand",
            icon: FileImportIcon,
            onSelect: () => {
              setView("import");
            },
          },
        ]
      : []),
    ...(grab.actions.cancel
      ? [
          "separator" as const,
          {
            label: "Cancel Grab",
            icon: Cancel01Icon,
            tone: "danger" as const,
            onSelect: () => {
              setView("cancel");
            },
          },
        ]
      : []),
  ];
  return (
    <>
      {torrent === undefined ? (
        <GrabMenu work={work} items={itemsWith([])} />
      ) : (
        <TorrentMenu {...job} row={torrent} itemsWith={itemsWith} />
      )}
      <GrabDialogs {...job} view={view} onView={setView} />
    </>
  );
}
