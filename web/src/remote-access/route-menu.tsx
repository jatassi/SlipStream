import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { ReachabilityExposureView } from "../api/gen/model";
import { removeExposure } from "../api/gen/query/reachability/reachability";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { PublicDialog } from "./public-dialog";
import { runAgain } from "./run-again";
import { useRefresh } from "./use-refresh";

function itemsOf(
  exposure: ReachabilityExposureView,
  edit: () => void,
  remove: () => void,
): MenuEntry[] {
  return [
    { label: "Run again", onSelect: () => void runAgain(exposure) },
    ...(exposure.kind === "media-server"
      ? [{ label: "Change public address…", onSelect: edit }]
      : []),
    "separator",
    { label: "Stop checking…", tone: "danger", onSelect: remove },
  ];
}

export function RouteMenu({ exposure }: { exposure: ReachabilityExposureView }) {
  const refresh = useRefresh();
  const [editing, setEditing] = useState(false);
  const [removing, setRemoving] = useState(false);
  const label = `${exposure.subject} actions`;
  return (
    <>
      <Menu
        label={label}
        trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
        items={itemsOf(
          exposure,
          () => {
            setEditing(true);
          },
          () => {
            setRemoving(true);
          },
        )}
      />
      <PublicDialog
        id={exposure.id}
        subject={exposure.subject}
        current={exposure.public}
        open={editing}
        onOpenChange={setEditing}
        onSaved={refresh}
      />
      <ConfirmDialog
        title={`Stop checking ${exposure.subject}?`}
        confirmLabel="Stop checking"
        look="fail"
        open={removing}
        onOpenChange={setRemoving}
        onConfirm={async () => {
          await removeExposure(exposure.id);
          await refresh();
        }}
      >
        <p>
          SlipStream stops checking whether players outside your home can connect to{" "}
          {exposure.subject}, and clears its Health item.
        </p>
      </ConfirmDialog>
    </>
  );
}
