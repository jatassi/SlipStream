import { useQuery } from "@tanstack/react-query";

import type {
  ConversionsConversionList,
  ConversionsConversionRow,
  FilesConvertingCell,
} from "../../api/gen/model";
import {
  getListConversionsQueryKey,
  listConversions,
} from "../../api/gen/query/conversions/conversions";
import { spanWords } from "../../format";
import { useLiveConversion } from "../../live/conversions";
import { stageTerms, pausedWords, waitWords } from "../../conversions/words";
import { useClock } from "../../ui/clock";
import { Data } from "../../ui/data";
import { ProgressCell } from "../../ui/progress";

const rowOf = (list: ConversionsConversionList, id: string): ConversionsConversionRow | undefined =>
  [
    ...list.attention,
    ...list.converting,
    ...list.next.flatMap(({ rows }) => rows.filter((row) => row.kind === "conversion")),
  ].find((row) => row.id === id);

function Progress({ row }: { row: ConversionsConversionRow }) {
  const live = useLiveConversion(row.id);
  const progress = live ?? row.progress;
  const paused = live?.paused ?? row.stage === "paused";
  const node = row.node?.name ?? "";
  return (
    <>
      {paused && row.pausedBy !== null && (
        <span className="text-meta text-ink-2">{pausedWords(row.pausedBy, node)}</span>
      )}
      {progress !== null && (
        <ProgressCell
          label={`Converting ${row.target.name}`}
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
      )}
    </>
  );
}

/**
 * What a converting cell adds under its chip: the Conversion's stage and
 * Node, then why it waits or how far it has run, from the Work's Conversions.
 */
export function ConvertingFacts({ cell, workId }: { cell: FilesConvertingCell; workId: string }) {
  const clock = useClock();
  const params = { work: workId };
  const row = useQuery({
    queryKey: getListConversionsQueryKey(params),
    queryFn: ({ signal }) => listConversions(params, { signal }),
    select: (list) => rowOf(list, cell.conversionId),
  }).data;
  const on = cell.node === null ? "" : ` on ${cell.node.name}`;
  return (
    <>
      <span className="text-meta text-ink-2">
        {stageTerms[cell.stage].word}
        {on}
      </span>
      {row !== undefined && row.wait !== null && (
        <span className="text-meta text-ink-3">
          {waitWords[row.wait.code](row.wait, row, clock)}
        </span>
      )}
      {row !== undefined && (row.stage === "running" || row.stage === "paused") && (
        <Progress row={row} />
      )}
    </>
  );
}
