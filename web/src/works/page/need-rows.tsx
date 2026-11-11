import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type {
  FilesUpgradeNeed,
  FilesWaitingNeedCode,
  GrabsClientWait,
  GrabsGrab,
} from "../../api/gen/model";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { SendToSheet } from "../../grabs/send-to-sheet";
import { Button } from "../../ui/button";
import { List, ListRow } from "../../ui/list";
import { WarningLine } from "../../ui/warning-line";
import { type Naming, upgradeWords } from "./need-words";

export function Upgrades({
  naming,
  needs,
}: {
  naming: Naming;
  needs: readonly FilesUpgradeNeed[];
}) {
  const search = useStartSearch();
  return (
    <List label="Upgrades available">
      {needs.map((need) => (
        <ListRow
          key={`${need.itemIds.join(" ")} ${need.edition}`}
          title={upgradeWords(naming, need)}
          meta="A better release would replace the file."
          trailing={
            <span className="flex gap-2">
              <Button
                size="small"
                busy={search.isPending}
                onClick={() => {
                  search.start(naming.work.id, { levelIds: [], itemIds: need.itemIds });
                }}
              >
                Search
              </Button>
              <ManualSearchButton
                workId={naming.work.id}
                scope={{ itemIds: need.itemIds }}
                size="small"
              />
            </span>
          }
        />
      ))}
    </List>
  );
}

/** A waiting Grab that the Admin can send to another client. */
function SendableWait({
  grab,
  subject,
  detail,
}: {
  grab: GrabsGrab;
  subject: string;
  detail: string;
}) {
  const [sending, setSending] = useState(false);
  return (
    <>
      <WarningLine
        signal="warn"
        chip="Waiting"
        subject={subject}
        detail={detail}
        action={{
          label: "Send to another client",
          onClick: () => {
            setSending(true);
          },
        }}
      />
      <SendToSheet grab={grab} open={sending} onOpenChange={setSending} />
    </>
  );
}

type Words = { readonly subject: string; readonly detail: string };

const routeWaitWords = ({ client, off }: GrabsClientWait, node: string): Words => {
  if (off) {
    return {
      subject: `Waiting for ${client === null ? "the built-in client" : client.name} on ${node}`,
      detail: `${client === null ? "The built-in client" : client.name} on ${node} is off. Turn it on or send this Grab to another client.`,
    };
  }
  return {
    subject: `Waiting to send to ${client === null ? "the built-in client" : client.name} on ${node}`,
    detail: `${node} is offline. This download goes there once ${node} is back online.`,
  };
};

/**
 * The words of a wait the Admin can end by sending the Grab elsewhere: to
 * fetch for its routed client's offline Node or its pinned client that is
 * off, or on its away client.
 */
function sendableWords(
  grab: GrabsGrab,
  code: FilesWaitingNeedCode,
  nodeName: (id: string) => string | undefined,
): Words | undefined {
  const { waitsFor, downloadClient } = grab;
  if (waitsFor !== undefined) {
    return routeWaitWords(waitsFor, nodeName(waitsFor.nodeId) ?? waitsFor.nodeId);
  }
  const node = nodeName(grab.nodeId);
  if (code !== "client_away" || downloadClient === null || node === undefined) {
    return undefined;
  }
  return {
    subject: `Waiting for ${downloadClient.name} on ${node}`,
    detail: `Its download carries on once ${downloadClient.name} responds again.`,
  };
}

/**
 * A Grab waiting on its Node that is away, on its client that isn't
 * responding, or to send its fetch to a client whose Node is offline.
 */
export function Waiting({
  grab,
  code,
}: {
  grab: GrabsGrab | undefined;
  code: FilesWaitingNeedCode;
}) {
  const navigate = useNavigate();
  const { nodes } = useListNodesSuspense().data;
  const node = nodes.find(({ id }) => id === grab?.nodeId);
  const release = grab?.releaseTitle ?? "This Work's download";
  const sendable =
    grab && sendableWords(grab, code, (nodeId) => nodes.find(({ id }) => id === nodeId)?.name);
  if (grab !== undefined && sendable !== undefined) {
    return <SendableWait grab={grab} {...sendable} />;
  }
  if (node === undefined) {
    return (
      <WarningLine
        signal="warn"
        chip="Waiting"
        subject={release}
        detail="This download has stopped until an Admin acts. Activity shows its stage."
        action={{
          label: "Open Activity",
          onClick: () => void navigate({ to: "/activity" }),
        }}
      />
    );
  }
  return (
    <WarningLine
      signal="warn"
      chip="Waiting"
      subject={`Waiting for ${node.name}`}
      detail={`Its download carries on once ${node.name} is back online.`}
      action={{
        label: `Open ${node.name}`,
        onClick: () => void navigate({ to: "/nodes/$nodeId", params: { nodeId: node.id } }),
      }}
    />
  );
}

export function NoRoot() {
  const navigate = useNavigate();
  return (
    <WarningLine
      signal="warn"
      chip="No root"
      subject="No library root for this Work"
      detail="Its downloads wait until it has one. Add a library root, or choose one in Edit."
      action={{
        label: "Open Library roots",
        onClick: () =>
          void navigate({
            to: "/settings/$section",
            params: { section: "library-roots" },
          }),
      }}
    />
  );
}
