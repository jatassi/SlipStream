import type { TransfersTransfer } from "../api/gen/model";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { LiveBar } from "../ui/live-bar";
import { SignalChip } from "../ui/signal-chip";
import { headline, legOf, paceLine, percentOf, routeWords } from "./words";

const signalOf = {
  failed: <SignalChip kind="fail">Failed</SignalChip>,
  needs_space: <SignalChip kind="warn">Needs space</SignalChip>,
} as const;
export function TransferRow({ transfer, nodeId }: { transfer: TransfersTransfer; nodeId: string }) {
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
