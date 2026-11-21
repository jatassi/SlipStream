import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { DownloadclientsClientCard as Card } from "../api/gen/model";
import { bytes, countWords, durationWords } from "../format";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { rateWords } from "../ui/setting-rate";
import { ClientVerdict, listenWords } from "./client-verdict";
import { RoutedHere } from "./routed-here";

const torrents = { one: "torrent", many: "torrents" } as const;

const limitWords = (perSecond: number): ReactNode =>
  perSecond === 0 ? "No limit" : <Data>{rateWords(perSecond)}</Data>;

function Machinery({ card: { client, limits, totals } }: { card: Card }) {
  return (
    <Facts look="well">
      {client !== undefined && (
        <>
          <Fact label="Listen address">{listenWords(client.listen)}</Fact>
          <Fact label="Peer ID prefix">
            <Data>{client.peerIdPrefix}</Data>
          </Fact>
          <Fact label="User agent">
            <Data>{client.userAgent}</Data>
          </Fact>
          <Fact label="libtorrent">
            <Data>{client.libtorrent}</Data>
          </Fact>
        </>
      )}
      <Fact label="Download limit">{limitWords(limits.downloadLimit)}</Fact>
      <Fact label="Upload limit">{limitWords(limits.uploadLimit)}</Fact>
      <Fact label="Downloads at once">
        {limits.activeDownloads === 0 ? "No limit" : limits.activeDownloads}
      </Fact>
      <Fact label="File checks at once">{limits.activeChecks}</Fact>
      <Fact label="Peer connections">{limits.connections}</Fact>
      <Fact label="uTP">{limits.utp ? "On" : "Off"}</Fact>
      <Fact label="Stalled after">
        <Data>{durationWords(limits.stalledAfterSeconds)}</Data>
      </Fact>
      <Fact label="Seeding time">
        <Data>{durationWords(totals.seedingSeconds)}</Data>
      </Fact>
      <Fact label="Active time">
        <Data>{durationWords(totals.activeSeconds)}</Data>
      </Fact>
    </Facts>
  );
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
          <Machinery card={card} />
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
        <p>This card disappears if this Node loses the Downloading Role.</p>
      </footer>
    </article>
  );
}
