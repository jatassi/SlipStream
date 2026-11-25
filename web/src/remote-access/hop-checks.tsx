import type { ReachabilityCheckView, ReachabilityHopView } from "../api/gen/model";
import { useClock } from "../ui/clock";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { InspectorSection } from "../ui/inspector";
import { whenWords } from "../ui/time-words";
import { FixText } from "./fix";
import { familyNames, observedLabels, results } from "./words";

function resultWords(check: ReachabilityCheckView): string {
  const result = results[check.result];
  return check.family === undefined ? result : `${result} over ${familyNames[check.family]}`;
}

function Check({ check, judgedAt }: { check: ReachabilityCheckView; judgedAt: string | null }) {
  const clock = useClock();
  const pointing = !check.fact && (check.result === "fails" || check.result === "not-tested");
  return (
    <InspectorSection title={resultWords(check)}>
      <div className="flex flex-col items-start gap-2">
        {check.result === "passes" && <p>{check.meaning}</p>}
        {check.found !== undefined && <p>{check.found}</p>}
        {pointing && check.fix !== null && <FixText fix={check.fix} />}
        {(check.observed !== undefined || check.by !== undefined) && (
          <Details>
            <Facts look="well">
              {check.observed !== undefined && (
                <Fact label={observedLabels[check.id]}>{check.observed}</Fact>
              )}
              {check.by !== undefined && (
                <Fact label="Seen by">
                  {judgedAt === null ? check.by : `${check.by} ${whenWords(judgedAt, "at", clock)}`}
                </Fact>
              )}
            </Facts>
          </Details>
        )}
      </div>
    </InspectorSection>
  );
}

/** A hop's facts, then each check leading with what its passing means, its fix or what it found, with its values under Details. */
export function HopChecks({
  hop,
  judgedAt,
}: {
  hop: ReachabilityHopView;
  judgedAt: string | null;
}) {
  return (
    <>
      {hop.facts.map((fact) => (
        <p key={fact.id}>{fact.text}</p>
      ))}
      {hop.checks.map((check) => (
        <Check key={`${check.id}:${check.family ?? ""}`} check={check} judgedAt={judgedAt} />
      ))}
    </>
  );
}
