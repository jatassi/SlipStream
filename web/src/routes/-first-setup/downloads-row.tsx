import { useQuery } from "@tanstack/react-query";

import { isProblem } from "../../api/errors";
import type { DownloadclientsClientCard as Card, SystemNodeConfig } from "../../api/gen/model";
import { getGetNodeConfigQueryKey, getNodeConfig } from "../../api/gen/query/system/system";
import { bytes } from "../../format";
import { NodeSettingField, PendingRestart, settingOf } from "../../nodes/node-setting";
import { Button } from "../../ui/button";
import { SignalChip } from "../../ui/signal-chip";
import { useNode } from "../_app/nodes/-node";

function Apart() {
  return (
    <output className="flex flex-col items-start gap-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">Different drive</SignalChip>
      <span>
        Downloads are on a different drive from the library, so every import copies the whole file.
        Move the downloads folder to the library&apos;s drive to make imports instant.
      </span>
    </output>
  );
}

const offline = (name: string) =>
  `${name} is offline. Its downloads folder shows here when it reconnects.`;

function Folder({ nodeId, config }: { nodeId: string; config: SystemNodeConfig }) {
  const node = useNode(nodeId);
  const field = settingOf(config, "torrent.downloads", "folder");
  return (
    <>
      <PendingRestart node={node} />
      <NodeSettingField nodeId={nodeId} field={field} saved="Saved the downloads folder" />
    </>
  );
}

function DownloadsFolder({ card }: { card: Card }) {
  const config = useQuery({
    queryKey: getGetNodeConfigQueryKey(card.nodeId),
    queryFn: ({ signal }) => getNodeConfig(card.nodeId, { signal }),
    retryOnMount: false,
  });
  if (config.data !== undefined) {
    return <Folder nodeId={card.nodeId} config={config.data} />;
  }
  if (config.error === null) {
    return null;
  }
  if (isProblem(config.error, "node_offline")) {
    return <p>{offline(card.nodeName)}</p>;
  }
  return (
    <div className="flex flex-col items-start gap-2">
      <p role="alert">
        {card.nodeName}&apos;s downloads folder didn&apos;t load. Try again, or skip this step and
        set it later in Nodes.
      </p>
      <Button
        size="small"
        onClick={() => {
          void config.refetch();
        }}
      >
        Try again
      </Button>
    </div>
  );
}

export function DownloadsRow({ card, apart }: { card: Card; apart: boolean }) {
  return (
    <li aria-label={card.nodeName} className="flex flex-col gap-3 px-4 py-3">
      <span className="font-semibold text-ink-1">{card.nodeName}</span>
      {apart && <Apart />}
      {card.inStep ? (
        <DownloadsFolder card={card} />
      ) : (
        <p>{offline(card.nodeName)}</p>
      )}
      {card.downloads !== undefined && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {card.downloads.name} has {bytes(card.downloads.free)} free.
        </p>
      )}
    </li>
  );
}
