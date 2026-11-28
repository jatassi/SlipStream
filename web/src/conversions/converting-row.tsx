import type { ConversionsConversionRow } from "../api/gen/model";
import { spanWords } from "../format";
import { useLiveConversion } from "../live/conversions";
import { Data } from "../ui/data";
import { ProgressCell } from "../ui/progress";
import { Mark } from "../ui/state-mark";
import { Lines, WorkLink, rowLook, subjectOf } from "./row-parts";
import { pausedWords, stageTerms } from "./words";

/** A running or paused Conversion, its numbers from conversions.live once a sample lands. */
export function ConvertingRow({ row }: { row: ConversionsConversionRow }) {
  const live = useLiveConversion(row.id);
  const paused = live?.paused ?? row.stage === "paused";
  const progress = live ?? row.progress;
  const node = row.node?.name ?? "";
  return (
    <div className={rowLook}>
      <Lines
        first={<WorkLink row={row} />}
        second={
          <>
            {row.target.name} on {node} · <Mark mark={stageTerms[paused ? "paused" : row.stage]} />
            {paused && row.pausedBy !== null && ` · ${pausedWords(row.pausedBy, node)}`}
          </>
        }
      />
      {progress !== null && (
        <span className="w-60 max-w-[45%]">
          <ProgressCell
            label={`Converting ${subjectOf(row)}`}
            value={Math.floor(progress.percent)}
            motion={paused ? "settled" : "live"}
            note={
              paused ? undefined : (
                <Data>
                  {spanWords(progress.secondsLeft)} left · {progress.speed.toFixed(1)}×
                </Data>
              )
            }
          />
        </span>
      )}
    </div>
  );
}
