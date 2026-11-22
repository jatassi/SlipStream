import { useState } from "react";

import type { DownloadclientsClientCard as Card, MovesMoveAsk, MovesSide } from "../api/gen/model";
import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { countWords } from "../format";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";
import { PreviewDialog } from "./preview-dialog";

/** A Download Client a Move names: its Node, and the external client there, absent for the built-in one. */
export type Side = { readonly nodeId: string; readonly clientId?: string | undefined };

const builtIn = "built-in";

const sideKey = ({ clientId }: Side) => clientId ?? builtIn;

const sameSide = (a: Side, b: Side) => a.nodeId === b.nodeId && a.clientId === b.clientId;

/** The clients on a Node a torrent can move into: its built-in client, then each external one on and responding. */
const clientsOn = (card: Card) => [
  { value: builtIn, label: "Built-in client" },
  ...card.externals
    .filter((client) => client.on && !client.away)
    .map((client) => ({ value: client.id, label: client.name })),
];

/** Where a torrent can go other than from: the first Node's built-in client, else the next one. */
export function firstOther(cards: readonly Card[], from: Side): Side | undefined {
  for (const card of cards) {
    for (const { value } of clientsOn(card)) {
      const side = { nodeId: card.nodeId, ...(value !== builtIn && { clientId: value }) };
      if (!sameSide(side, from)) {
        return side;
      }
    }
  }
  return undefined;
}

export const sideBody = (side: Side): MovesSide => ({
  nodeId: side.nodeId,
  ...(side.clientId !== undefined && { clientId: side.clientId }),
});

/** A Node, then a Download Client on it. */
export function ClientPicker({
  cards,
  value,
  onValueChange,
}: {
  cards: readonly Card[];
  value: Side;
  onValueChange: (side: Side) => void;
}) {
  const card = cards.find(({ nodeId }) => nodeId === value.nodeId);
  return (
    <div className="flex flex-wrap gap-3">
      <Select
        label="Node"
        options={cards.map(({ nodeId, nodeName }) => ({ value: nodeId, label: nodeName }))}
        value={value.nodeId}
        onValueChange={(nodeId) => {
          onValueChange({ nodeId });
        }}
      />
      {card !== undefined && (
        <Select
          label="Download Client"
          options={clientsOn(card)}
          value={sideKey(value)}
          onValueChange={(key) => {
            onValueChange({ nodeId: value.nodeId, ...(key !== builtIn && { clientId: key }) });
          }}
        />
      )}
    </div>
  );
}

type Props = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/** Moves torrents to a Download Client on any Node, after a preview that gives each one a verdict. */
export function TorrentMoveDialog({
  torrents,
  open,
  onOpenChange,
}: Props & { readonly torrents: readonly ({ id: string; name: string } & Side)[] }) {
  const { cards } = useListDownloadClientsSuspense().data;
  const [first] = torrents;
  const [to, setTo] = useState(first && firstOther(cards, first));
  const ask: MovesMoveAsk | undefined = to && {
    origin: "torrent",
    works: [],
    torrents: torrents.map(({ id }) => ({ torrentId: id, ...sideBody(to) })),
  };
  return (
    <PreviewDialog
      title={
        torrents.length === 1 && first !== undefined
          ? `Move ${first.name}`
          : `Move ${countWords(torrents.length, { one: "torrent", many: "torrents" })}`
      }
      previewing={ask && { kind: "ask", ask }}
      picker={to && <ClientPicker cards={cards} value={to} onValueChange={setTo} />}
      startedWords={(started) =>
        `Moving ${countWords(started, { one: "torrent", many: "torrents" })}.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

/** Moves every torrent of one Download Client to another, and offers to send its Trackers' new Grabs there too. */
export function ClientTorrentsDialog({
  name,
  from,
  open,
  onOpenChange,
}: Props & { readonly name: string; readonly from: Side }) {
  const { cards } = useListDownloadClientsSuspense().data;
  const [to, setTo] = useState(firstOther(cards, from));
  const [repoint, setRepoint] = useState(true);
  const ask: MovesMoveAsk | undefined = to && {
    origin: "client",
    works: [],
    clients: [{ ...sideBody(from), to: sideBody(to) }],
    options: { repointRouting: repoint },
  };
  return (
    <PreviewDialog
      title={`Move the torrents of ${name}`}
      previewing={ask && { kind: "ask", ask }}
      picker={
        to && (
          <>
            <ClientPicker cards={cards} value={to} onValueChange={setTo} />
            <Checkbox
              label="Send each Tracker's new Grabs there too"
              checked={repoint}
              onCheckedChange={setRepoint}
            />
          </>
        )
      }
      startedWords={(started) =>
        `Moving ${countWords(started, { one: "torrent", many: "torrents" })} from ${name}.`
      }
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
