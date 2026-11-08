import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { Chip } from "../ui/chip";
import { SignalChip } from "../ui/signal-chip";
import { statusOf } from "./client-status";

/** An external client's status in words: its chip, then what happened and what to do. */
export function StatusLine({ client, kind, node }: { client: External; kind: Kind; node: string }) {
  const { signal, chip, sentence } = statusOf(client, kind, node);
  return (
    <div className="flex flex-col items-start gap-1.5">
      {signal === undefined ? <Chip>{chip}</Chip> : <SignalChip kind={signal}>{chip}</SignalChip>}
      {sentence !== undefined && <p className="text-pretty text-ink-1">{sentence}</p>}
    </div>
  );
}
