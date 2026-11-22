import type { TransfersTransfer } from "../api/gen/model";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { ProgressCell, percentOf } from "../ui/progress";
import { Mark } from "../ui/state-mark";
import { transferTerms } from "./terms";
import { headline, legOf, paceLine, routeWords } from "./words";

export function TransferRow({ transfer, nodeId }: { transfer: TransfersTransfer; nodeId: string }) {
  const leg = legOf(transfer, nodeId);
  const copying = transfer.state === "copying";
  return (
    <li className="flex flex-col gap-1 px-4 py-3 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
        <span className="min-w-0 truncate font-semibold text-ink-1">{transfer.jobName}</span>
        {transfer.purpose === "move" && <span className="text-meta text-ink-3">Move</span>}
        {!copying && (
          <>
            <span className="text-meta text-ink-3">
              {leg.sends ? `To ${leg.peer}` : `From ${leg.peer}`}
            </span>
            <Mark mark={transferTerms[transfer.state]} />
          </>
        )}
      </div>
      <Details
        summary={headline(transfer, leg)}
        action={
          copying && (
            <span className="mt-1 block w-full">
              <ProgressCell
                label={`Transferring ${transfer.jobName}`}
                value={percentOf(transfer.verified, transfer.total, 100)}
                motion="live"
                span="line"
                note={<Data>{paceLine(transfer)}</Data>}
              />
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
  return (
    <div className="mt-1">
      <Facts look="well">
        {route !== undefined && <Fact label="Route">{routeWords(route, transfer)}</Fact>}
        <Fact label="Lanes">{details.lanes}</Fact>
        <Fact label="Round trip">
          <Data>{details.rttMs} ms</Data>
        </Fact>
        <Fact label="Retried chunks">{details.retried}</Fact>
        <Fact label="Files">{`${verified} of ${files.length} verified`}</Fact>
      </Facts>
    </div>
  );
}
