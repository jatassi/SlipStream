import { useState } from "react";

import { ApiError, isProblem } from "../../api/errors";
import type { SettingField as Field, WorksContentType, WorksWorkView } from "../../api/gen/model";
import { setWorkOptions } from "../../api/gen/query/works/works";
import { countWords } from "../../format";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { SettingField } from "../../ui/setting-field";
import { Switch } from "../../ui/switch";
import { lowered } from "../monitoring/labels";

/** A previewed change to one option, waiting on the Admin's yes. */
type Pending = {
  readonly field: Field;
  readonly value: unknown;
  readonly withdraws: number;
};

function pickedWords(field: Field, value: unknown): string {
  if (field.kind !== "choice") {
    return String(value);
  }
  const option = field.options.find((each) => each.value === value);
  if (option === undefined) {
    throw new Error(`${field.key} offers no ${String(value)}`);
  }
  return option.label;
}

const unavailableWords = (type: WorksContentType, { field, value }: Omit<Pending, "withdraws">) =>
  `${type.source} doesn't list this ${lowered(type.work).one} in ${pickedWords(field, value)}. Choose another ${field.label.toLowerCase()}.`;

/** The source's refusal of a value, in words that name the value. */
function refusalOf(type: WorksContentType, error: unknown, pending: Omit<Pending, "withdraws">) {
  if (!isProblem(error, "order_unavailable")) {
    return error;
  }
  return new ApiError({
    status: error.status,
    code: error.code,
    title: error.title,
    detail: unavailableWords(type, pending),
    errors: [...error.errors],
  });
}

function outerLevels(type: WorksContentType) {
  const [level] = type.levels;
  if (level === undefined) {
    throw new Error(`${type.id} has per-Work options and no level`);
  }
  return lowered(level);
}

function ChangeWords({ type, pending }: { type: WorksContentType; pending: Pending }) {
  const items = lowered(type.item);
  const { withdraws } = pending;
  return (
    <>
      <p>
        Changing the {pending.field.label.toLowerCase()} moves {items.many} between{" "}
        {outerLevels(type).many}. Each {items.one} keeps its files, Grabs and monitoring.
      </p>
      {withdraws > 0 && (
        <p>
          {`${countWords(withdraws, items)} ${withdraws === 1 ? "isn't" : "aren't"} in ${pickedWords(pending.field, pending.value)}, so ${withdraws === 1 ? "it'll" : "they'll"} show as No longer listed by ${type.source}.`}
        </p>
      )}
      <p>
        File names change to the new numbering when SlipStream renames files. Renaming comes in a
        later release.
      </p>
    </>
  );
}

function useOptionChange(type: WorksContentType, work: WorksWorkView) {
  const [options, setOptions] = useState(work.options);
  const [pending, setPending] = useState<Pending>();
  const send = (field: Field, value: unknown, preview: boolean) =>
    setWorkOptions(work.id, { type: { [field.key]: value }, preview });
  return {
    options,
    pending,
    cancel: () => {
      setPending(undefined);
    },
    preview: async (field: Field, value: unknown) => {
      try {
        setPending({ field, value, withdraws: (await send(field, value, true)).withdraws });
      } catch (error) {
        throw refusalOf(type, error, { field, value });
      }
    },
    apply: async ({ field, value }: Pending) => {
      try {
        setOptions((await send(field, value, false)).options);
      } catch (error) {
        const refusal = refusalOf(type, error, { field, value });
        throw refusal instanceof ApiError ? new Error(refusal.detail ?? refusal.title) : refusal;
      }
    },
  };
}

/** The Work's level folders switch, "Volume folders", for a type whose Items sit in levels. */
export function LevelFolders({
  type,
  on,
  onChange,
}: {
  type: WorksContentType;
  on: boolean;
  onChange: (on: boolean) => void;
}) {
  const [level] = type.levels;
  if (type.items === "one" || level === undefined) {
    return null;
  }
  return <Switch label={`${level.one} folders`} checked={on} onCheckedChange={onChange} />;
}

export const saveLevelFolders = (workId: string, levelFolders: boolean) =>
  setWorkOptions(workId, { levelFolders, preview: false });

/**
 * The type's options for this Work. A change is previewed first, since it
 * can move Items between levels, and saved once the Admin confirms it.
 */
export function OptionsFields({ type, work }: { type: WorksContentType; work: WorksWorkView }) {
  const { options, pending, cancel, preview, apply } = useOptionChange(type, work);
  return (
    <>
      {options.map((field) => (
        <SettingField
          key={field.key}
          field={field}
          onSave={(value) => preview(field, value ?? field.default)}
        />
      ))}
      {pending !== undefined && (
        <ConfirmDialog
          title={`Change the ${pending.field.label.toLowerCase()} to ${pickedWords(pending.field, pending.value)}?`}
          confirmLabel={`Change ${pending.field.label.toLowerCase()}`}
          open
          onOpenChange={(open) => {
            if (!open) {
              cancel();
            }
          }}
          onConfirm={() => apply(pending)}
        >
          <ChangeWords type={type} pending={pending} />
        </ConfirmDialog>
      )}
    </>
  );
}
