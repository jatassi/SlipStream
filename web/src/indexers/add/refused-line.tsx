import { Details } from "../../ui/details";
import { SignalLine } from "../../ui/signal-line";

/** A refusal, with the reason a definition doesn't load under Details. */
export type Refused = { readonly said: string; readonly reason: string | undefined };

export function RefusedLine({ refused }: { refused: Refused }) {
  const said = (
    <SignalLine tone="fail" announce>
      {refused.said}
    </SignalLine>
  );
  return refused.reason === undefined ? (
    said
  ) : (
    <div className="flex flex-col items-start gap-1">
      {said}
      <Details>
        <p className="mt-1 max-w-[64ch]">{refused.reason}</p>
      </Details>
    </div>
  );
}
