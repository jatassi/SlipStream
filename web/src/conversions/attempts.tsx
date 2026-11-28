import type { ConversionsAttemptOutcome, ConversionsConversionRow } from "../api/gen/model";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";

const outcomeWords: { readonly [O in ConversionsAttemptOutcome | "running"]: string } = {
  running: "Running",
  failed: "Failed",
  stalled: "Stalled",
  disk_full: "The disk was full",
  put_back: "Stopped, and put back in line",
};

export function Attempts({ row }: { row: ConversionsConversionRow }) {
  return (
    <Details>
      <Facts look="failure">
        {row.attempts.map((attempt) => (
          <Fact key={attempt.n} label={`Attempt ${attempt.n} on ${attempt.node}`}>
            {outcomeWords[attempt.outcome ?? "running"]}
            {attempt.code !== "" && (
              <>
                {" · "}
                <Data>{attempt.code}</Data>
              </>
            )}
          </Fact>
        ))}
        {row.failure !== null && row.failure.detail !== "" && (
          <Fact label="ffmpeg">
            <Data className="whitespace-pre-wrap wrap-anywhere">{row.failure.detail}</Data>
          </Fact>
        )}
      </Facts>
    </Details>
  );
}
