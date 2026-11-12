import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { SettingListField, SettingListFieldItem } from "../api/gen/model";
import { type Noun, countWords } from "../format";
import { Chip } from "./chip";
import { Icon } from "./icon";
import {
  type ControlProps,
  type Kind,
  SaveButton,
  defaultHint,
  labelLook,
  onSubmit,
} from "./setting-kind";

type Written = { entry: string } | { refused: string };

const items: {
  readonly [I in SettingListFieldItem]: {
    noun: Noun;
    adding: string;
    write: (typed: string) => Written;
  };
} = {
  extension: {
    noun: { one: "extension", many: "extensions" },
    adding: "Add an extension",
    write: (typed) => {
      const entry = typed.trim().replace(/^\./u, "").toLowerCase();
      if (entry === "") {
        return { refused: "An extension can't be blank." };
      }
      return /^[\da-z]+$/u.test(entry)
        ? { entry }
        : {
            refused: `"${typed.trim()}" isn't an extension. Use letters and digits only, such as mkv.`,
          };
    },
  },
};

const separators = /[\s,]+/u;

const fewest = (field: SettingListField) =>
  `Keep at least ${countWords(field.min, items[field.item].noun)} in the list.`;

function useEntries(
  field: SettingListField,
  submit: ControlProps<"list">["submit"],
  onEdit: () => void,
) {
  const item = items[field.item];
  const [entries, setEntries] = useState(field.value);
  const [typed, setTyped] = useState("");
  const add = (text: string): string[] | undefined => {
    const next = [...entries];
    for (const piece of text.split(separators).filter((part) => part !== "")) {
      const written = item.write(piece);
      if ("refused" in written) {
        submit(written);
        return undefined;
      }
      if (next.includes(written.entry)) {
        submit({ refused: `${written.entry} is already in the list.` });
        return undefined;
      }
      next.push(written.entry);
    }
    setEntries(next);
    setTyped("");
    return next;
  };
  const changed =
    typed.trim() !== "" ||
    entries.length !== field.value.length ||
    entries.some((entry, index) => entry !== field.value[index]);
  const type = (next: string) => {
    onEdit();
    if (!separators.test(next) || add(next) === undefined) {
      setTyped(next);
    }
  };
  const remove = (entry: string) => {
    setEntries(entries.filter((kept) => kept !== entry));
    onEdit();
  };
  const removeLast = () => {
    const last = entries.at(-1);
    if (last !== undefined) {
      remove(last);
    }
  };
  return { item, entries, typed, add, type, remove, removeLast, changed };
}

function Entry({
  entry,
  disabled,
  onRemove,
}: {
  entry: string;
  disabled: boolean;
  onRemove: () => void;
}) {
  return (
    <li className="contents">
      <Chip>
        {entry}
        <button
          type="button"
          aria-label={`Remove ${entry}`}
          disabled={disabled}
          onClick={onRemove}
          className="-mr-1 inline-flex size-4 items-center justify-center rounded-capsule text-ink-3 transition-colors duration-(--duration-control) ease-standard hover:text-ink-1 disabled:opacity-40 in-data-[density=touch]:size-6"
        >
          <Icon icon={Cancel01Icon} className="size-3" />
        </button>
      </Chip>
    </li>
  );
}

function Entries({
  label,
  entries,
  disabled,
  onRemove,
}: {
  label: string;
  entries: readonly string[];
  disabled: boolean;
  onRemove: (entry: string) => void;
}) {
  return entries.length === 0 ? null : (
    <ul aria-label={label} className="contents">
      {entries.map((entry) => (
        <Entry
          key={entry}
          entry={entry}
          disabled={disabled}
          onRemove={() => {
            onRemove(entry);
          }}
        />
      ))}
    </ul>
  );
}

function EntryInput({
  placeholder,
  typed,
  onType,
  onAdd,
  onRemoveLast,
}: {
  placeholder: string;
  typed: string;
  onType: (next: string) => void;
  onAdd: () => void;
  onRemoveLast: () => void;
}) {
  return (
    <Input
      autoComplete="off"
      spellCheck={false}
      placeholder={placeholder}
      value={typed}
      onValueChange={onType}
      onKeyDown={(event) => {
        if (event.key === "Enter" && typed.trim() !== "") {
          event.preventDefault();
          onAdd();
        } else if (event.key === "Backspace" && typed === "") {
          onRemoveLast();
        }
      }}
      className="h-7 min-w-32 flex-1 bg-transparent px-1.5 text-body text-ink-1 outline-none placeholder:text-ink-3 focus-visible:shadow-none disabled:opacity-40 in-data-[density=touch]:text-input-touch"
    />
  );
}

/** Entries as chips in one well, with a text input after them that adds an entry on Enter, a comma or a space. */
function ListControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"list">) {
  const { item, entries, typed, add, type, remove, removeLast, changed } = useEntries(
    field,
    submit,
    onEdit,
  );
  return (
    <form
      onSubmit={onSubmit(() => {
        const next = add(typed);
        if (next !== undefined) {
          submit(next.length < field.min ? { refused: fewest(field) } : { value: next });
        }
      })}
      className="flex flex-col items-start gap-1.5"
    >
      <Field.Root
        invalid={invalid}
        disabled={disabled}
        className="flex flex-col gap-1.5 self-stretch"
      >
        <Field.Label className={labelLook}>{field.label}</Field.Label>
        <div className="flex min-h-(--input-height) flex-wrap items-center gap-1.5 rounded-item bg-well px-1.5 py-1 ring-1 ring-hairline ring-inset has-aria-invalid:ring-fail-line has-focus-visible:shadow-[0_0_0_6px_var(--color-halo)] has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-white">
          <Entries label={field.label} entries={entries} disabled={disabled} onRemove={remove} />
          <EntryInput
            placeholder={item.adding}
            typed={typed}
            onType={type}
            onAdd={() => {
              add(typed);
            }}
            onRemoveLast={removeLast}
          />
        </div>
      </Field.Root>
      <SaveButton shown={changed} busy={busy} />
    </form>
  );
}

export const listKind: Kind<"list"> = {
  words: (field) => field.value.join(", "),
  hints: (field) => [
    [
      defaultHint(field, field.default.join(", ")),
      ...(field.min === 0 ? [] : [`At least ${countWords(field.min, items[field.item].noun)}.`]),
    ].join(" "),
  ],
  Control: ListControl,
};
