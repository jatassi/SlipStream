import type { ReactNode } from "react";

import type { FormatsExplained, FormatsParsed } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import {
  conditionNotes,
  conditionResult,
  factRows,
  formatGroups,
  formatResult,
  verdictWords,
} from "./tester-words";

function Part({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <h3 className="font-semibold text-ink-1">{title}</h3>
      {children}
    </div>
  );
}

function Facts({ result }: { result: FormatsParsed }) {
  const rows = factRows(result.facts);
  return (
    <Part title="What it reads as">
      {rows.length === 0 ? (
        <p>The name gives no facts SlipStream reads.</p>
      ) : (
        <dl className="grid self-start grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
          {rows.map(({ label, value }) => (
            <div key={label} className="contents">
              <dt className="text-ink-3">{label}</dt>
              <dd className="wrap-anywhere">{label === "Titles" ? value : <Data>{value}</Data>}</dd>
            </div>
          ))}
        </dl>
      )}
    </Part>
  );
}

function Format({ format }: { format: FormatsExplained }) {
  return (
    <li className="flex flex-col gap-1">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="font-semibold text-ink-1">{format.name}</span>
        {format.draft && <Chip>Draft</Chip>}
        <span>{formatResult(format)}</span>
      </div>
      <ul aria-label={`Conditions of ${format.name}`} className="flex flex-col gap-0.5 pl-4">
        {format.conditions.map((condition, index) => (
          // A format can hold two conditions of one name.
          <li key={`${String(index)} ${condition.name}`} className="flex flex-wrap gap-x-2">
            <span className="text-ink-1">{condition.name}</span>
            <span>{conditionResult(condition)}</span>
            {conditionNotes(condition).map((note) => (
              <span key={note} className="text-ink-3">
                {note}
              </span>
            ))}
          </li>
        ))}
      </ul>
    </li>
  );
}

function Formats({ formats }: { formats: readonly FormatsExplained[] }) {
  const { shown, others } = formatGroups(formats);
  return (
    <Part title="Custom Formats">
      {formats.length === 0 && <p>No Custom Formats yet.</p>}
      {shown.length > 0 && (
        <ul aria-label="Custom Formats it matches" className="flex flex-col gap-2">
          {shown.map((format) => (
            <Format key={format.id} format={format} />
          ))}
        </ul>
      )}
      {others.length > 0 && <p>Doesn&apos;t match: {others.join(", ")}.</p>}
    </Part>
  );
}

function Targets({ result }: { result: FormatsParsed }) {
  return (
    <Part title="Targets">
      {result.targets.length === 0 ? (
        <p>No Target is set up.</p>
      ) : (
        <ul aria-label="Targets" className="flex flex-col gap-2">
          {result.targets.map((target) => (
            <li key={target.targetId} className="flex flex-col gap-0.5">
              <div className="flex flex-wrap gap-x-3">
                <span className="font-semibold text-ink-1">{target.name}</span>
                <span className="tabular-nums">Score {target.score}</span>
                <span>{verdictWords(result, target.wouldFill)}</span>
              </div>
              {target.reasons.length > 0 && (
                <ul className="flex flex-col gap-0.5 pl-4">
                  {target.reasons.map((reason) => (
                    <li key={`${reason.code} ${reason.value ?? ""}`}>{reason.words}</li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      )}
    </Part>
  );
}

export function TestedName({ result }: { result: FormatsParsed }) {
  return (
    <li className="border-t border-hairline py-3 first:border-t-0 first:pt-0">
      <Details
        summary={result.sentence}
        action={<Data className="wrap-anywhere text-ink-3">{result.name}</Data>}
      >
        <div className="mt-1 flex flex-col gap-4">
          <Facts result={result} />
          <Formats formats={result.formats} />
          <Targets result={result} />
        </div>
      </Details>
    </li>
  );
}
