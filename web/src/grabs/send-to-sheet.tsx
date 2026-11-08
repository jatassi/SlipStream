import { Suspense, useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  DownloadclientsClientCard as Card,
  GrabsGrab,
  SendToInputBody,
} from "../api/gen/model";
import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useSendGrabTo } from "../api/gen/query/grabs/grabs";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { RadioGroup } from "../ui/radio";
import { Skeleton } from "../ui/skeleton";
import { toast } from "../ui/toast";

/** A client a Grab can go to: its label, its name in a sentence, and the send-to body that names it. */
export type ClientPick = {
  readonly key: string;
  readonly label: string;
  readonly name: string;
  readonly body: SendToInputBody;
};

export const builtInKey = (nodeId: string) => `built-in:${nodeId}`;

export const currentKey = (grab: GrabsGrab): string =>
  grab.downloadClient?.id ?? builtInKey(grab.nodeId);

/**
 * Every client that can take a Grab: each downloading Node's built-in
 * client, then its external clients that are on and responding.
 */
export const clientsOf = (cards: readonly Card[]): ClientPick[] =>
  cards.flatMap((card) => [
    {
      key: builtInKey(card.nodeId),
      label: `Built-in client on ${card.nodeName}`,
      name: `the built-in client on ${card.nodeName}`,
      body: { builtInOn: card.nodeId },
    },
    ...card.externals
      .filter((client) => client.on && !client.away)
      .map((client) => ({
        key: client.id,
        label: `${client.name} on ${card.nodeName}`,
        name: `${client.name} on ${card.nodeName}`,
        body: { clientId: client.id },
      })),
  ]);

export function ClientChoice({
  picks,
  value,
  onValueChange,
}: {
  picks: readonly ClientPick[];
  value: string;
  onValueChange: (key: string) => void;
}) {
  return (
    <RadioGroup
      label="Send to"
      options={picks.map(({ key, label }) => ({ value: key, label }))}
      value={value}
      onValueChange={onValueChange}
    />
  );
}

function SendTo({ grab, onDone }: { grab: GrabsGrab; onDone: () => void }) {
  const { cards } = useListDownloadClientsSuspense().data;
  const picks = clientsOf(cards).filter(({ key }) => key !== currentKey(grab));
  const [picked, setPicked] = useState(picks[0]?.key ?? "");
  const send = useSendGrabTo();
  const chosen = picks.find(({ key }) => key === picked);
  if (chosen === undefined) {
    return (
      <div className="flex flex-col gap-4">
        <p>No other client can take this download. Add a Download Client in Settings first.</p>
        <div className="flex justify-end">
          <Button onClick={onDone}>Close</Button>
        </div>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-4">
      <p>SlipStream sends {grab.releaseTitle} to the client you choose.</p>
      <ClientChoice picks={picks} value={picked} onValueChange={setPicked} />
      {send.isError && <p className="text-fail-ink">{sentenceOf(send.error)}</p>}
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onDone}>Cancel</Button>
        <Button
          look="primary"
          busy={send.isPending}
          onClick={() => {
            send.mutate(
              { grabId: grab.id, data: chosen.body },
              {
                onSuccess: () => {
                  toast(`Sent ${grab.releaseTitle} to ${chosen.name}`);
                  onDone();
                },
              },
            );
          }}
        >
          Send
        </Button>
      </div>
    </div>
  );
}

/** Sends a Grab's download to another client, chosen from every client that can take it. */
export function SendToSheet({
  grab,
  open,
  onOpenChange,
}: {
  grab: GrabsGrab;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Send to another client" open={open} onOpenChange={onOpenChange}>
      <Suspense fallback={<Skeleton label="Loading the clients" />}>
        <SendTo
          grab={grab}
          onDone={() => {
            onOpenChange(false);
          }}
        />
      </Suspense>
    </Dialog>
  );
}
