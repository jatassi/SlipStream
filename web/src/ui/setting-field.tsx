import { Alert02Icon, ArrowDown01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { UnreachableError, isApiError } from "../api/errors";
import { Button } from "./button";
import { Icon } from "./icon";
import { Popover } from "./popover";
import type { Fields, SettingKind, Values } from "./setting-kind";
import { kinds } from "./setting-kinds";
import { Sheet, SheetCancel } from "./sheet";
import { useShell } from "./shell-context";

export type { Setting, SettingKind } from "./setting-kind";

/** Saves a value, or null to reset to the default; rejects with the server's refusal. */
export type SettingSave<K extends SettingKind> = (value: Values[K] | null) => Promise<unknown>;

function refusal(error: unknown): string {
  if (isApiError(error)) {
    return error.detail ?? error.title;
  }
  if (error instanceof UnreachableError) {
    return error.message;
  }
  throw error;
}

function useSave<K extends SettingKind>(onSave: SettingSave<K>, onSaved: () => void) {
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const send = async (value: Values[K] | null) => {
    setBusy(true);
    setError(undefined);
    try {
      await onSave(value);
      onSaved();
    } catch (failure: unknown) {
      setError(refusal(failure));
    } finally {
      setBusy(false);
    }
  };
  return { error, setError, busy, send };
}

function Refusal({ error }: { error: string }) {
  return (
    <p
      role="alert"
      className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
    >
      <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
      {error}
    </p>
  );
}

function Hints({ hints, reset }: { hints: readonly string[]; reset: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
      <div className="flex min-w-0 flex-col gap-0.5 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {hints.map((hint) => (
          <p key={hint}>{hint}</p>
        ))}
      </div>
      {reset}
    </div>
  );
}

type BodyProps<K extends SettingKind> = {
  kind: K;
  field: Fields[K];
  onSave: SettingSave<K>;
  onSaved: () => void;
};

function Body<K extends SettingKind>({ kind, field, onSave, onSaved }: BodyProps<K>) {
  const { error, setError, busy, send } = useSave(onSave, onSaved);
  const { Control, hints } = kinds[kind];
  return (
    <div className="flex flex-col gap-2">
      <Control
        key={JSON.stringify(field.value)}
        field={field}
        busy={busy}
        invalid={error !== undefined}
        submit={(draft) => {
          if ("refused" in draft) {
            setError(draft.refused);
          } else {
            void send(draft.value);
          }
        }}
        onEdit={() => {
          setError(undefined);
        }}
      />
      {error !== undefined && <Refusal error={error} />}
      <Hints
        hints={hints(field)}
        reset={
          !field.isDefault && (
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
          )
        }
      />
    </div>
  );
}

const rowLook =
  "-mx-2 inline-flex min-h-(--control-height) max-w-full items-center gap-1.5 rounded-item px-2 py-1 text-left text-pretty text-ink-1 transition-colors duration-(--duration-control) ease-standard hover:bg-fill data-popup-open:bg-fill";

const withUnitsKept = (words: string): string => words.replaceAll(/(?<=\d) (?=\S)/gu, "\u00A0");

function Row<K extends SettingKind>({
  words,
  ...body
}: Omit<BodyProps<K>, "onSaved"> & { words: string }) {
  const [open, setOpen] = useState(false);
  const trigger = (
    <button type="button" aria-label={`${body.field.label}: ${words}`} className={rowLook}>
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
    <Sheet trigger={trigger} title={body.field.label} open={open} onOpenChange={setOpen}>
      <div className="rounded-card bg-fill p-4">{full}</div>
      <SheetCancel />
    </Sheet>
  ) : (
    <Popover trigger={trigger} label={body.field.label} open={open} onOpenChange={setOpen}>
      {full}
    </Popover>
  );
}

const ignoreSaved = () => {};

export type SettingFieldProps<K extends SettingKind> = {
  field: Fields[K] & { kind: K };
  onSave: SettingSave<K>;
} & ({ variant?: "full" } | { variant: "row"; words?: string | undefined });

/**
 * The one control for an Admin setting of any kind. `full` edits it in
 * place; `row` shows the value in words and opens the full field. Each
 * owner passes its own save, so the field never knows where it is stored.
 */
export function SettingField<K extends SettingKind>(props: SettingFieldProps<K>) {
  const kind: K = props.field.kind;
  if (props.variant === "row") {
    return (
      <Row
        kind={kind}
        field={props.field}
        onSave={props.onSave}
        words={props.words ?? kinds[kind].words(props.field)}
      />
    );
  }
  return <Body kind={kind} field={props.field} onSave={props.onSave} onSaved={ignoreSaved} />;
}

export function SettingGroup({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col rounded-card bg-glass ring-1 ring-hairline ring-inset *:px-4 *:py-3.5 *:not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      {children}
    </div>
  );
}
