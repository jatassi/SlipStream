import type {
  TransfersTransfer,
  TransfersTransferRoute,
  TransfersTransferState,
} from "../api/gen/model";
import { bytes, countWords } from "../format";
import { leftWords, rateWords } from "../torrents/model";
import type { StatusPart } from "../ui/status-sentence";

/** One Node's view of a Transfer: whether it sends or receives, and the Node at the other end. */
export type Leg = { readonly sends: boolean; readonly peer: string };

export const legOf = (transfer: TransfersTransfer, nodeId: string): Leg =>
  transfer.source.nodeId === nodeId
    ? { sends: true, peer: transfer.destination.nodeName }
    : { sends: false, peer: transfer.source.nodeName };

export const stateLabel: { readonly [S in TransfersTransferState]: string } = {
  queued: "Queued",
  waiting: "Waiting",
  needs_space: "Needs space",
  copying: "Transferring",
  failed: "Failed",
};

/**
 * What a Transfer is doing, in one sentence: "Transferring from seedbox" while
 * it copies, else the Controller's reason word for word. A leg names the far
 * Node from that Node's side; without one it names the source.
 */
export function headline(transfer: TransfersTransfer, leg?: Leg): string {
  if (transfer.state !== "copying") {
    return transfer.reason ?? stateLabel[transfer.state];
  }
  if (leg?.sends === true) {
    return `Transferring to ${leg.peer}`;
  }
  return `Transferring from ${leg?.peer ?? transfer.source.nodeName}`;
}

/** "2.1 GB of 8.4 GB · 48 MB/s · 2 min left", leaving out a speed or estimate not known yet. */
export const paceLine = ({ verified, total, rate, estimateSeconds }: TransfersTransfer): string =>
  [
    `${bytes(verified)} of ${bytes(total)}`,
    rate > 0 && rateWords(rate),
    estimateSeconds > 0 && leftWords(estimateSeconds),
  ]
    .filter((part) => part !== false)
    .join(" · ");

export const percentOf = ({ verified, total }: TransfersTransfer): number =>
  total === 0 ? 100 : Math.floor((verified / total) * 100);

const transferNoun = { one: "Transfer", many: "Transfers" };

function movingPart(copying: readonly TransfersTransfer[], nodeId: string, sends: boolean) {
  const [only] = copying;
  if (only === undefined) {
    return [];
  }
  const rate = copying.reduce((sum, transfer) => sum + transfer.rate, 0);
  const pace = rate > 0 ? ` at ${rateWords(rate)}` : "";
  const verb = sends ? "Sending" : "Receiving";
  const what =
    copying.length === 1
      ? `${only.jobName} ${sends ? "to" : "from"} ${legOf(only, nodeId).peer}`
      : countWords(copying.length, transferNoun);
  return [{ kind: "live", text: `${verb} ${what}${pace}` } as const];
}

/**
 * The sentence that opens a Node's Transfers: what it sends and receives now,
 * then how many wait and how many failed, or "Nothing moving".
 */
export function statusOf(
  transfers: readonly TransfersTransfer[],
  nodeId: string,
  failedHref: string,
): StatusPart[] {
  const copying = transfers.filter(({ state }) => state === "copying");
  const waiting = transfers.filter(({ state }) => state !== "copying" && state !== "failed");
  const failed = transfers.filter(({ state }) => state === "failed");
  const parts: StatusPart[] = [
    ...movingPart(
      copying.filter((transfer) => legOf(transfer, nodeId).sends),
      nodeId,
      true,
    ),
    ...movingPart(
      copying.filter((transfer) => !legOf(transfer, nodeId).sends),
      nodeId,
      false,
    ),
    ...(waiting.length > 0 ? [{ kind: "plain", text: `${waiting.length} waiting` } as const] : []),
    ...(failed.length > 0
      ? [
          {
            kind: "failure",
            text: countWords(failed.length, { one: "failure", many: "failures" }),
            href: failedHref,
          } as const,
        ]
      : []),
  ];
  return parts.length > 0 ? parts : [{ kind: "plain", text: "Nothing moving" }];
}

/** Which Node opens the lanes, as a Details value. */
export function routeWords(route: TransfersTransferRoute, transfer: TransfersTransfer): string {
  const source = transfer.source.nodeName;
  const destination = transfer.destination.nodeName;
  switch (route) {
    case "receiver_dials":
      return `${destination} dials ${source}`;
    case "sender_dials":
      return `${source} dials ${destination}`;
    case "node_link":
      return "The Node link";
    default:
      return route satisfies never;
  }
}
