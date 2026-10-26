import { useQueryClient } from "@tanstack/react-query";

import type { NodesNode, TransfersTransfer } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import {
  setTransferSettings,
  useGetTransferSettingsSuspense,
  useListTransferSpeedsSuspense,
  useListTransfersSuspense,
} from "../api/gen/query/transfers/transfers";
import { clock } from "../format";
import { adoptTransferSettings } from "../live/adopt";
import { rateWords } from "../torrents/model";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { List, ListRow } from "../ui/list";
import { LiveBar } from "../ui/live-bar";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { SignalChip } from "../ui/signal-chip";
import { StatusSentence } from "../ui/status-sentence";
import { toast } from "../ui/toast";
import { headline, legOf, paceLine, percentOf, routeWords, statusOf } from "./words";

const listId = "transfers";

/** Nodes > a Node > Transfers: what the Node sends and receives, its send limit and its recent speeds. */
export function NodeTransfers({ node }: { node: NodesNode }) {
  const { transfers } = useListTransfersSuspense({ nodeId: node.id }).data;
  return (
    <>
      <div className="mb-6">
        <StatusSentence parts={statusOf(transfers, node.id, `#${listId}`)} />
      </div>
      <SendLimit nodeId={node.id} />
      {transfers.length > 0 && (
        <Section id={listId} title="Transfers" count={transfers.length}>
          <ul
            aria-label={`Transfers of ${node.name}`}
            className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
          >
            {transfers.map((transfer) => (
              <TransferRow key={transfer.id} transfer={transfer} nodeId={node.id} />
            ))}
          </ul>
        </Section>
      )}
      <Speeds node={node} />
    </>
  );
}

function SendLimit({ nodeId }: { nodeId: string }) {
  const { sendLimit } = useGetTransferSettingsSuspense(nodeId).data;
  const queryClient = useQueryClient();
  return (
    <Section title="Settings">
      <SettingGroup>
        <SettingField
          field={sendLimit}
          onSave={async (value) => {
            adoptTransferSettings(
              queryClient,
              nodeId,
              await setTransferSettings(nodeId, { sendLimit: value }),
            );
            toast("Transfer settings saved");
          }}
        />
      </SettingGroup>
    </Section>
  );
}

const signalOf = {
  failed: <SignalChip kind="fail">Failed</SignalChip>,
  needs_space: <SignalChip kind="warn">Needs space</SignalChip>,
} as const;

function TransferRow({ transfer, nodeId }: { transfer: TransfersTransfer; nodeId: string }) {
  const leg = legOf(transfer, nodeId);
  const copying = transfer.state === "copying";
  return (
    <li className="flex flex-col gap-1 px-4 py-3 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
        <span className="min-w-0 truncate font-semibold text-ink-1">{transfer.jobName}</span>
        {!copying && (
          <span className="text-meta text-ink-3">
            {leg.sends ? `To ${leg.peer}` : `From ${leg.peer}`}
          </span>
        )}
        {(transfer.state === "failed" || transfer.state === "needs_space") &&
          signalOf[transfer.state]}
      </div>
      <Details
        summary={headline(transfer, leg)}
        action={
          copying && (
            <span className="mt-1 flex w-full max-w-[360px] flex-col gap-1.5">
              <Data className="text-meta text-ink-3">{paceLine(transfer)}</Data>
              <LiveBar label={`Transferring ${transfer.jobName}`} value={percentOf(transfer)} />
            </span>
          )
        }
      >
        <Machinery transfer={transfer} />
      </Details>
    </li>
  );
}

function Machinery({ transfer }: { transfer: TransfersTransfer }) {
  const { details, files, route } = transfer;
  const verified = files.filter((file) => file.verified).length;
  const facts = [
    ...(route === undefined ? [] : [{ label: "Route", value: routeWords(route, transfer) }]),
    { label: "Lanes", value: String(details.lanes) },
    { label: "Round trip", value: <Data>{details.rttMs} ms</Data> },
    { label: "Retried chunks", value: String(details.retried) },
    { label: "Files", value: `${verified} of ${files.length} verified` },
  ];
  return (
    <dl className="mt-1 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
      {facts.map(({ label, value }) => (
        <div key={label} className="contents">
          <dt className="text-ink-3">{label}</dt>
          <dd className="wrap-anywhere">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function Speeds({ node }: { node: NodesNode }) {
  const { speeds } = useListTransferSpeedsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const rows = speeds
    .flatMap((speed) => {
      const sends = speed.sourceNodeId === node.id;
      const peerId = sends ? speed.destinationNodeId : speed.sourceNodeId;
      const peer = nodes.find((other) => other.id === peerId);
      if ((!sends && speed.destinationNodeId !== node.id) || peer === undefined) {
        return [];
      }
      return [{ speed, title: sends ? `To ${peer.name}` : `From ${peer.name}` }];
    })
    .toSorted((a, b) => a.title.localeCompare(b.title));
  return (
    <Section title="Recent speeds">
      {rows.length === 0 ? (
        <p className="text-ink-3">
          No Transfer has moved bytes between {node.name} and another Node yet.
        </p>
      ) : (
        <List label={`Recent speeds of ${node.name}`}>
          {rows.map(({ speed, title }) => (
            <ListRow
              key={title}
              title={title}
              meta={`Measured ${clock(speed.rateAt)}`}
              trailing={<Data className="text-ink-1">{rateWords(speed.rate)}</Data>}
            />
          ))}
        </List>
      )}
    </Section>
  );
}
