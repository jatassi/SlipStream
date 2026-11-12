import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type {
  FilesClientError,
  FilesImportWait,
  FilesWaitingNeed,
  FilesWaitingNeedCode,
  GrabsClientWait,
  GrabsGrab,
} from "../../api/gen/model";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { useListRootsSuspense } from "../../api/gen/query/roots/roots";
import { SendToSheet } from "../../grabs/send-to-sheet";
import { rootWords } from "../../roots/words";
import { WarningLine } from "../../ui/warning-line";
import { type ImportWait, type Naming, clientErrorWords, importWaitWords } from "./need-words";

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

/** A Grab whose external client reports an error, until it fails. */
function ClientErrorWait({
  grab,
  release,
  error,
}: {
  grab: GrabsGrab | undefined;
  release: string;
  error: FilesClientError;
}) {
  const navigate = useNavigate();
  const torrentId = grab?.torrentId ?? null;
  return (
    <WarningLine
      signal="warn"
      chip="Waiting"
      subject={release}
      detail={clientErrorWords(error)}
      action={
        torrentId === null
          ? {
              label: "Open Activity",
              onClick: () => void navigate({ to: "/activity" }),
            }
          : {
              label: "Open torrent",
              onClick: () =>
                void navigate({
                  to: "/activity/torrents/$torrentId",
                  params: { torrentId },
                }),
            }
      }
    />
  );
}

/** A Grab whose Import waits on space, its root or reading a file. */
function ImportWaitLine({
  release,
  code,
  type,
  hold,
  nodeName,
}: {
  release: string;
  code: ImportWait;
  type: Naming["type"];
  hold: FilesImportWait | null;
  nodeName: (id: string) => string | undefined;
}) {
  const navigate = useNavigate();
  const { roots } = useListRootsSuspense().data;
  const root = roots.find(({ id }) => id === hold?.rootId);
  const named =
    root === undefined
      ? "library root"
      : rootWords(root, roots, type, nodeName(root.nodeId) ?? "a removed Node");
  return (
    <WarningLine
      signal="warn"
      chip="Waiting"
      subject={release}
      detail={importWaitWords(code, named, hold?.space ?? null)}
      action={{
        label: "Open Pending imports",
        onClick: () => void navigate({ to: "/library/pending" }),
      }}
    />
  );
}

const isImportWait = (code: FilesWaitingNeedCode): code is ImportWait =>
  code === "no_space" || code === "root_not_ready" || code === "probe_timeout";

/**
 * A Grab waiting on its Node that is away, on its client that isn't
 * responding or reports an error, to send its fetch to a client whose Node
 * is offline, or on its Import.
 */
export function Waiting({
  type,
  grab,
  need: { code, clientError, importWait },
}: {
  type: Naming["type"];
  grab: GrabsGrab | undefined;
  need: FilesWaitingNeed;
}) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodes.find(({ id }) => id === grab?.nodeId);
  const release = grab?.releaseTitle ?? "This Work's download";
  const nodeName = (nodeId: string) => nodes.find(({ id }) => id === nodeId)?.name;
  const sendable = grab && sendableWords(grab, code, nodeName);
  if (grab !== undefined && sendable !== undefined) {
    return <SendableWait grab={grab} {...sendable} />;
  }
  if (code === "client_error" && clientError !== null) {
    return <ClientErrorWait grab={grab} release={release} error={clientError} />;
  }
  if (isImportWait(code)) {
    return (
      <ImportWaitLine
        release={release}
        code={code}
        type={type}
        hold={importWait}
        nodeName={nodeName}
      />
    );
  }
  return <NodeWait node={node} release={release} />;
}

/** A Grab waiting on its Node that is away, or stopped for another cause. */
function NodeWait({
  node,
  release,
}: {
  node: { id: string; name: string } | undefined;
  release: string;
}) {
  const navigate = useNavigate();
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
