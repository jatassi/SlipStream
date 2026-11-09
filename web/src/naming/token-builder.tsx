import { useState } from "react";

import type { NamingToken, NamingTokenGroup, WorksContentType } from "../api/gen/model";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Dialog } from "../ui/dialog";
import { TokenControls } from "./token-controls";
import { type TokenChoice, plainChoice, tokenText } from "./token-text";

const groupOrder: readonly NamingTokenGroup[] = [
  "work",
  "level",
  "item",
  "target",
  "release",
  "media",
];

const groupLabels = (type: WorksContentType): Readonly<Record<NamingTokenGroup, string>> => ({
  work: type.work.one,
  level: type.levels.map((level) => level.one).join(" and "),
  item: type.item.one,
  target: "Target and cut",
  release: "Release",
  media: "Media",
});

function TokenGroups({
  type,
  tokens,
  picked,
  onPick,
}: {
  type: WorksContentType;
  tokens: readonly NamingToken[];
  picked: NamingToken | undefined;
  onPick: (token: NamingToken) => void;
}) {
  const labels = groupLabels(type);
  return groupOrder.map((group) => {
    const grouped = tokens.filter((token) => token.group === group);
    return grouped.length === 0 ? null : (
      <fieldset key={group} className="flex flex-col gap-1.5">
        <legend className="mb-1.5 text-label text-ink-3">{labels[group]}</legend>
        <div className="flex flex-wrap gap-1.5">
          {grouped.map((token) => (
            <button
              key={token.name}
              type="button"
              aria-pressed={token === picked}
              onClick={() => {
                onPick(token);
              }}
              className="h-7 rounded-capsule bg-fill px-3 text-chip text-ink-2 ring-1 ring-hairline ring-inset hover:bg-fill-hover hover:text-ink-1 aria-pressed:bg-selected aria-pressed:text-ink-1"
            >
              {token.name}
            </button>
          ))}
        </div>
      </fieldset>
    );
  });
}

function Picked({
  token,
  choice,
  onChange,
}: {
  token: NamingToken;
  choice: TokenChoice;
  onChange: (choice: TokenChoice) => void;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-card bg-well p-4 ring-1 ring-hairline ring-inset">
      <p className="text-body text-ink-2">
        {token.about} For example, &quot;{token.example}&quot;.
      </p>
      <TokenControls token={token} choice={choice} onChange={onChange} />
    </div>
  );
}

function InsertRow({
  text,
  onInsert,
}: {
  text: string | undefined;
  onInsert: (text: string) => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <output aria-label="Token text" className="min-w-0 break-all text-ink-1">
        {text === undefined ? (
          <span className="text-ink-3">Pick a token.</span>
        ) : (
          <Data>{text}</Data>
        )}
      </output>
      <Button
        look="primary"
        disabled={text === undefined}
        onClick={() => {
          if (text !== undefined) {
            onInsert(text);
          }
        }}
      >
        Insert
      </Button>
    </div>
  );
}

/**
 * The tokens a template may name, by group. Picking one shows what it prints,
 * the controls it offers and the text Insert puts at the caret.
 */
export function TokenBuilder({
  type,
  label,
  tokens,
  onInsert,
}: {
  type: WorksContentType;
  label: string;
  tokens: readonly NamingToken[];
  onInsert: (text: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<NamingToken>();
  const [choice, setChoice] = useState(plainChoice);
  const text = picked === undefined ? undefined : tokenText(picked, choice);
  return (
    <Dialog
      title={`${label} tokens`}
      size={680}
      open={open}
      onOpenChange={setOpen}
      trigger={
        <Button size="small" aria-label={`Tokens for ${label}`}>
          Tokens
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        <TokenGroups
          type={type}
          tokens={tokens}
          picked={picked}
          onPick={(token) => {
            setPicked(token);
            setChoice(plainChoice);
          }}
        />
        {picked !== undefined && <Picked token={picked} choice={choice} onChange={setChoice} />}
        <InsertRow
          text={text}
          onInsert={(inserted) => {
            setOpen(false);
            onInsert(inserted);
          }}
        />
      </div>
    </Dialog>
  );
}
