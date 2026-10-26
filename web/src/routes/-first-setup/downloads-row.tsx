import type { DownloadclientsClientCard as Card } from "../../api/gen/model";
import { useGetNodeConfigSuspense } from "../../api/gen/query/system/system";
import { bytes } from "../../format";
import { NodeSettingField, PendingRestart, settingOf } from "../../nodes/node-setting";
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

function DownloadsFolder({ nodeId }: { nodeId: string }) {
  const node = useNode(nodeId);
  const field = settingOf(useGetNodeConfigSuspense(nodeId).data, "torrent.downloads", "folder");
  return (
    <>
      <PendingRestart node={node} />
      <NodeSettingField nodeId={nodeId} field={field} saved="Saved the downloads folder" />
    </>
  );
}

export function DownloadsRow({ card, apart }: { card: Card; apart: boolean }) {
  return (
    <li aria-label={card.nodeName} className="flex flex-col gap-3 px-4 py-3">
      <span className="font-semibold text-ink-1">{card.nodeName}</span>
      {apart && <Apart />}
      {card.inStep ? (
        <DownloadsFolder nodeId={card.nodeId} />
      ) : (
        <p>{card.nodeName} is offline. Its downloads folder shows here when it reconnects.</p>
      )}
      {card.downloads !== undefined && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {bytes(card.downloads.free)} free on {card.downloads.name}.
        </p>
      )}
    </li>
  );
}
