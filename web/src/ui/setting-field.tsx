import { ArrowDown01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { Button } from "./button";
import { Icon } from "./icon";
import { Popover } from "./popover";
import type { Fields, SettingKind, Values } from "./setting-kind";
import { kinds } from "./setting-kinds";
import { Hints, Lock, Refusal, appliesHint, useSave } from "./setting-parts";
import { Sheet, SheetCancel } from "./sheet";
import { useShell } from "./shell-context";

export type { Setting, SettingKind } from "./setting-kind";

/** Saves a value, or null to reset to the default; rejects with the server's refusal. */
export type SettingSave<K extends SettingKind> = (
  value: Values[K] | null,
) => Promise<unknown>;

/** Answers what saving a value would change that the Admin should see first, or undefined for nothing. */
export type SettingWarn<K extends SettingKind> = (
  value: Values[K],
) => Promise<string | undefined>;

type BodyProps<K extends SettingKind> = {
  kind: K;
  field: Fields[K];
  onSave: SettingSave<K>;
  onWarn?: SettingWarn<K> | undefined;
  onSaved: () => void;
};

function ResetButton({
  busy,
  send,
}: {
  busy: boolean;
  send: (value: null) => Promise<void>;
}) {
  return (
    <Button
      look="ghost"
      size="small"
      disabled={busy}
      onClick={() => {
        void send(null);
      }}
    >
      Reset to default
    </Button>
  );
}

function Warned({
  words,
  busy,
  onSave,
}: {
  words: string;
  busy: boolean;
  onSave: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <output className="text-meta text-ink-2 in-data-[density=touch]:text-body-touch">
        {words}
      </output>
      <Button look="ghost" size="small" disabled={busy} onClick={onSave}>
        Save anyway
      </Button>
    </div>
  );
}

function useWarned<V>(
  onWarn: ((value: V) => Promise<string | undefined>) | undefined,
  send: (value: V) => Promise<void>,
  busy: boolean,
) {
  const [warned, setWarned] = useState<{ value: V; words: string }>();
  const save = async (value: V) => {
    const words = onWarn === undefined ? undefined : await onWarn(value);
    if (words === undefined) {
      await send(value);
    } else {
      setWarned({ value, words });
    }
  };
  const notice = warned !== undefined && (
    <Warned
      words={warned.words}
      busy={busy}
      onSave={() => {
        setWarned(undefined);
        void send(warned.value);
      }}
    />
  );
  const clear = () => {
    setWarned(undefined);
  };
  return { save, notice, clear };
}

function Body<K extends SettingKind>({
  kind,
  field,
  onSave,
  onWarn,
  onSaved,
}: BodyProps<K>) {
  const { error, setError, busy, send } = useSave(onSave, onSaved);
  const { save, notice, clear } = useWarned(onWarn, send, busy);
  const { Control, hints } = kinds[kind];
  const locked = field.lockedBy;
  const edit = () => {
    setError(undefined);
    clear();
  };
  return (
    <div className="flex flex-col gap-2">
      <Control
        key={JSON.stringify(field.value)}
        field={field}
        busy={busy}
        invalid={error !== undefined}
        disabled={locked !== undefined}
        submit={(draft) => {
          if ("refused" in draft) {
            setError(draft.refused);
          } else {
            void save(draft.value);
          }
        }}
        onEdit={edit}
      />
      {error !== undefined && <Refusal error={error} />}
      {notice}
      {locked !== undefined && <Lock variable={locked} />}
      <Hints
        hints={[...hints(field), ...appliesHint[field.applies]]}
        reset={
          !field.isDefault &&
          locked === undefined && <ResetButton busy={busy} send={send} />
        }
      />
    </div>
  );
}

const rowLook =
  "-mx-2 inline-flex min-h-(--control-height) max-w-[calc(100%+1rem)] items-center gap-1.5 rounded-item px-2 py-1 text-left text-pretty text-ink-1 transition-colors duration-(--duration-control) ease-standard hover:bg-fill data-popup-open:bg-fill";

const withUnitsKept = (words: string): string =>
  words.replaceAll(/(?<=\d) (?=\S)/gu, "\u00A0");

function Row<K extends SettingKind>({
  words,
  ...body
}: Omit<BodyProps<K>, "onSaved"> & { words: string }) {
  const [open, setOpen] = useState(false);
  const trigger = (
    <button
      type="button"
      aria-label={`${body.field.label}: ${words}`}
      className={rowLook}
    >
      <span>{withUnitsKept(words)}</span>
      <Icon icon={ArrowDown01Icon} className="size-3.5 shrink-0 text-ink-3" />
    </button>
  );
  const full = (
    <Body
      {...body}
      onSaved={() => {
        setOpen(false);
      }}
    />
  );
  return useShell() === "phone" ? (
    <Sheet
      trigger={trigger}
      title={body.field.label}
      open={open}
      onOpenChange={setOpen}
    >
      <div className="rounded-card bg-fill p-4">{full}</div>
      <SheetCancel />
    </Sheet>
  ) : (
    <Popover
      trigger={trigger}
      label={body.field.label}
      open={open}
      onOpenChange={setOpen}
    >
      {full}
    </Popover>
  );
}

const ignoreSaved = () => {};

export type SettingFieldProps<K extends SettingKind> = {
  field: Fields[K] & { kind: K };
  onSave: SettingSave<K>;
  onWarn?: SettingWarn<K>;
} & ({ variant?: "full" } | { variant: "row"; words?: string | undefined });

/**
 * The one control for an Admin setting of any kind. `full` edits it in
 * place; `row` shows the value in words and opens the full field. Each
 * owner passes its own save, so the field never knows where it is stored.
 */
export function SettingField<K extends SettingKind>(
  props: SettingFieldProps<K>,
) {
  const kind: K = props.field.kind;
  if (props.variant === "row") {
    return (
      <Row
        kind={kind}
        field={props.field}
        onSave={props.onSave}
        onWarn={props.onWarn}
        words={props.words ?? kinds[kind].words(props.field)}
      />
    );
  }
  return (
    <Body
      kind={kind}
      field={props.field}
      onSave={props.onSave}
      onWarn={props.onWarn}
      onSaved={ignoreSaved}
    />
  );
}

export function SettingGroup({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col rounded-card bg-glass ring-1 ring-hairline ring-inset *:px-4 *:py-3.5 *:not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      {children}
    </div>
  );
}
