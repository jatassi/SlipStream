import type { ReactNode } from "react";

import type { DownloadclientsClientCard as Card } from "../api/gen/model";
import { Data } from "../ui/data";
import { SignalChip } from "../ui/signal-chip";
import { unspecified, verdictOf, verdictWords } from "./verdict";

/** The listen-port verdict a client card and the Torrent client section lead with. */
export function ClientVerdict({ card }: { card: Card }) {
  const { signal, chip, sentence, hint } = verdictWords(verdictOf(card), card.nodeName);
  return (
    <div className="flex flex-col items-start gap-1.5">
      {signal !== undefined && chip !== undefined && <SignalChip kind={signal}>{chip}</SignalChip>}
      <p className="text-ink-1 text-pretty">{sentence}</p>
      {hint !== undefined && <p className="text-ink-3 text-pretty">{hint}</p>}
    </div>
  );
}

export function listenWords(listen: string): ReactNode {
  return unspecified(listen) ? (
    <span>
      <Data>{listen}</Data>, the address this computer reaches the internet from
    </span>
  ) : (
    listen
  );
}
