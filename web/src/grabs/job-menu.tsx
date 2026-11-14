import {
  ArrowUpRight01Icon,
  Cancel01Icon,
  FileImportIcon,
  InformationCircleIcon,
  MoreHorizontalIcon,
} from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type { GrabsGrab } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { ActionsMenu } from "../torrents/actions-menu";
import { type TorrentAction, menuFor, subjectOf } from "../torrents/actions";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { type GrabView, GrabDialogs } from "./grab-details";
// oxlint-disable-next-line import/max-dependencies -- a Grab's menu wraps its Torrent's and adds Details, Import by hand and Cancel
import { byHand } from "./stages";

type Job = { readonly grab: GrabsGrab; readonly work: string };

/** A Grab's own menu items around its Torrent's: Details first, Cancel Grab last. */
type GrabItems = (torrent: readonly MenuEntry[]) => MenuEntry[];

const jobActions = new Set<TorrentAction>(["pause", "resume", "remove"]);

function TorrentMenu({
  grab,
  work,
  torrentId,
  itemsWith,
}: Job & { torrentId: string; itemsWith: GrabItems }) {
  const navigate = useNavigate();
  const row = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === torrentId) },
  }).data;
  if (row === undefined) {
    return <GrabMenu work={work} items={itemsWith([])} />;
  }
  const actions =
    grab.stage === "downloading" ? menuFor(row).filter((action) => jobActions.has(action)) : [];
  return (
    <ActionsMenu
      subject={subjectOf(row, work)}
      actions={actions}
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
export function JobMenu({ grab, work }: Job) {
  const [view, setView] = useState<GrabView>();
  const itemsWith: GrabItems = (torrent) => [
    {
      label: "Details",
      icon: InformationCircleIcon,
      onSelect: () => {
        setView("details");
      },
    },
    ...torrent,
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
      {grab.torrentId === null ? (
        <GrabMenu work={work} items={itemsWith([])} />
      ) : (
        <TorrentMenu grab={grab} work={work} torrentId={grab.torrentId} itemsWith={itemsWith} />
      )}
      <GrabDialogs grab={grab} work={work} view={view} onView={setView} />
    </>
  );
}
