import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { DownloadclientsClientCard as Card } from "../api/gen/model";
import { bytes, countWords, durationWords } from "../format";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Details } from "../ui/details";
import { rateWords } from "../ui/setting-rate";
import { ClientVerdict, listenWords } from "./client-verdict";
import { RoutedHere } from "./routed-here";

const torrents = { one: "torrent", many: "torrents" } as const;

const limitWords = (perSecond: number): ReactNode =>
  perSecond === 0 ? <span>No limit</span> : rateWords(perSecond);

const words = (text: string): ReactNode => <span>{text}</span>;

function rows(card: Card): (readonly [string, ReactNode])[] {
  const { client, limits, totals } = card;
  return [
    ...(client === undefined
      ? []
      : [
          ["Listen address", listenWords(client.listen)] as const,
          ["Peer ID prefix", client.peerIdPrefix] as const,
          ["User agent", client.userAgent] as const,
          ["libtorrent", client.libtorrent] as const,
        ]),
    ["Download limit", limitWords(limits.downloadLimit)],
    ["Upload limit", limitWords(limits.uploadLimit)],
    [
      "Downloads at once",
      words(limits.activeDownloads === 0 ? "No limit" : String(limits.activeDownloads)),
    ],
    ["File checks at once", words(String(limits.activeChecks))],
    ["Peer connections", words(String(limits.connections))],
    ["uTP", words(limits.utp ? "On" : "Off")],
    ["Stalled after", durationWords(limits.stalledAfterSeconds)],
    ["Seeding time", durationWords(totals.seedingSeconds)],
    ["Active time", durationWords(totals.activeSeconds)],
  ];
}

const totalsWords = ({ torrents: held, totals }: Card): string =>
  `${countWords(held, torrents)}, ${bytes(totals.uploaded)} uploaded and ${bytes(totals.downloaded)} downloaded.`;

/** A Node's built-in client: its verdict, its totals, and under Details its identity and limits. */
export function BuiltInClient({ card }: { card: Card }) {
  return (
    <article aria-label={`Built-in client on ${card.nodeName}`} className="flex flex-col gap-3">
      <h4 className="font-semibold text-ink-1">Built-in client</h4>
      <ClientVerdict card={card} />
      <RoutedHere trackers={card.trackers} />
      <Details summary={totalsWords(card)}>
        <div className="mt-2 self-stretch">
          <KeyValues rows={rows(card)} />
        </div>
      </Details>
      <footer className="flex flex-col gap-1 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        <Link
          to="/nodes/$nodeId/torrent-client"
          params={{ nodeId: card.nodeId }}
          className="text-ink-2 underline hover:text-ink-1"
        >
          Torrent client settings on {card.nodeName}
        </Link>
        <p>This card is removed once this Node stops downloading.</p>
      </footer>
    </article>
  );
}
