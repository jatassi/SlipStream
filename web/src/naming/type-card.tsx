import { useEffect, useId } from "react";

import type { NamingTypeNaming, WorksContentType } from "../api/gen/model";
import { useListNamingTokensSuspense } from "../api/gen/query/naming/naming";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { Refusal } from "../ui/setting-parts";
import { type Draft, cardOf } from "./card";
import { CardFields } from "./card-fields";
import { type Lead, recommend } from "./recommendation";
import { type Refusal as Refused, useNamingCard } from "./use-naming-card";

/** The lead's button, then a refused save or what the save check refuses in the stored settings. */
function LeadAction({
  lead,
  naming,
  refusal,
  busy,
  onSave,
}: {
  lead: Lead;
  naming: NamingTypeNaming;
  refusal: Refused;
  busy: boolean;
  onSave: (values: Readonly<Record<string, unknown>>) => void;
}) {
  const labels = new Map(naming.settings.map((setting) => [setting.key, setting.label]));
  const shown = refusal.length > 0 ? refusal : naming.problems;
  const { action } = lead;
  return (
    <div className="flex flex-col items-start gap-2 py-1">
      {action !== undefined && (
        <Button
          size="small"
          busy={busy}
          onClick={() => {
            onSave(action.values);
          }}
        >
          {action.label}
        </Button>
      )}
      {shown.length > 0 && (
        <ul className="flex flex-col gap-1">
          {shown.map(({ key, sentence }) => {
            const label = key === undefined ? undefined : labels.get(key);
            return (
              <li key={`${key}:${sentence}`}>
                <Refusal error={label === undefined ? sentence : `${label}: ${sentence}`} />
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

type TypeCardProps = {
  readonly type: WorksContentType;
  readonly naming: NamingTypeNaming;
  /** Takes the card's unsaved values after each change, for the file name tester to name under. */
  readonly onDraft: (draft: Draft) => void;
};

/**
 * One Content Type's naming. It leads with what to do and its button, and
 * Details holds the settings, edited as one draft and saved together.
 */
export function TypeCard({ type, naming, onDraft }: TypeCardProps) {
  const titleId = useId();
  const lead = recommend(type, naming);
  const { fields } = useListNamingTokensSuspense(type.id).data;
  const { draft, preview, refusal, busy, change, save, saveDraft, discard } = useNamingCard(naming);
  useEffect(() => {
    onDraft(draft);
  }, [onDraft, draft]);
  return (
    <article
      id={type.id}
      aria-labelledby={titleId}
      className="flex scroll-mt-20 flex-col gap-2 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <h3 id={titleId} className="text-card-title text-ink-1">
        {type.name}
      </h3>
      <Details
        fields
        summary={lead.sentence}
        action={
          <LeadAction
            lead={lead}
            naming={naming}
            refusal={refusal}
            busy={busy}
            onSave={(values) => void save(values)}
          />
        }
      >
        <CardFields
          type={type}
          card={cardOf(naming)}
          tokens={fields}
          draft={draft}
          preview={preview}
          busy={busy}
          onChange={change}
          onSave={() => void saveDraft()}
          onDiscard={discard}
        />
      </Details>
    </article>
  );
}
