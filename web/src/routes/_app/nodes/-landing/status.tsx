import { RemoteAccessLine } from "../../../../remote-access/summary-line";
import { StatusSentence } from "../../../../ui/status-sentence";

export function LandingStatus({ sentence }: { sentence: string }) {
  return (
    <div className="mb-4 flex flex-col gap-1">
      <StatusSentence parts={[{ kind: "plain", text: sentence }]} />
      <RemoteAccessLine />
    </div>
  );
}
