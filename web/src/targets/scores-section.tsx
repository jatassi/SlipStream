import { Alert02Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { Details } from "../ui/details";
import { Icon } from "../ui/icon";
import { SearchField } from "../ui/search-field";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";
import { DraftField } from "./field";
import {
  type ScoresDraft,
  incrementProblem,
  matching,
  scoreProblem,
  scoresInWords,
  withScore,
} from "./scores";

function Totals({
  scores,
  onScores,
}: {
  scores: ScoresDraft;
  onScores: (next: ScoresDraft) => void;
}) {
  return (
    <div className="grid gap-x-3 gap-y-4 @min-[620px]:grid-cols-3">
      <DraftField
        label="Minimum score"
        value={scores.minimum}
        error={scoreProblem(scores.minimum)}
        description="A Release scoring below it isn't grabbed."
        onChange={(minimum) => {
          onScores({ ...scores, minimum });
        }}
      />
      <DraftField
        label="Upgrade until"
        value={scores.upgradeUntil}
        error={scoreProblem(scores.upgradeUntil)}
        description="A file stops upgrading by score once it scores this."
        onChange={(upgradeUntil) => {
          onScores({ ...scores, upgradeUntil });
        }}
      />
      <DraftField
        label="Increment"
        value={scores.increment}
        error={incrementProblem(scores.increment)}
        description="How much more a Release must score to replace a file of its tier."
        onChange={(increment) => {
          onScores({ ...scores, increment });
        }}
      />
    </div>
  );
}

function FormatRow({
  name,
  score,
  onScore,
}: {
  name: string;
  score: string;
  onScore: (score: string) => void;
}) {
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_8rem] items-start gap-x-3 px-4 py-3">
      <span className="min-w-0 pt-7 font-semibold break-words text-ink-1">{name}</span>
      <DraftField
        label="Score"
        name={`${name} score`}
        value={score}
        error={scoreProblem(score)}
        onChange={onScore}
      />
    </li>
  );
}

function FormatScores({
  scores,
  onScores,
}: {
  scores: ScoresDraft;
  onScores: (next: ScoresDraft) => void;
}) {
  const [filter, setFilter] = useState("");
  if (scores.formats.length === 0) {
    return null;
  }
  const shown = matching(scores, filter);
  return (
    <div className="flex flex-col gap-3">
      <SearchField
        label="Filter Custom Formats"
        placeholder="Filter by name"
        value={filter}
        onValueChange={setFilter}
        className="max-w-[320px]"
      />
      {shown.length === 0 ? (
        <p className="text-ink-3">
          No Custom Format&apos;s name contains &quot;{filter.trim()}&quot;.
        </p>
      ) : (
        <ul
          aria-label="Custom Format scores"
          className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {shown.map(({ id, name, score }) => (
            <FormatRow
              key={id}
              name={name}
              score={score}
              onScore={(next) => {
                onScores(withScore(scores, id, next));
              }}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

export function ScoresSection({ draft, onChange }: SectionProps<TargetDraft>) {
  const { lead, warnings } = scoresInWords(draft.scores);
  const onScores = (scores: ScoresDraft) => {
    onChange({ ...draft, scores });
  };
  return (
    <Details
      fields
      summary={<span aria-live="polite">{lead}</span>}
      action={warnings.map((warning) => (
        <p key={warning} className="flex items-start gap-1.5 text-warn-ink">
          <Icon icon={Alert02Icon} className="mt-0.5 size-4 shrink-0" />
          {warning}
        </p>
      ))}
    >
      <div className="@container flex flex-col gap-6 pt-3">
        <Totals scores={draft.scores} onScores={onScores} />
        <FormatScores scores={draft.scores} onScores={onScores} />
      </div>
    </Details>
  );
}
