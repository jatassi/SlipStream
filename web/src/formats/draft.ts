import type {
  FormatsCondition,
  FormatsCustomFormat,
  FormatsCustomFormatDialect,
  FormatsNewFormat,
  FormatsOfferedKind,
  FormatsSchema,
} from "../api/gen/model";

export type Kind = FormatsOfferedKind;
export type Dialect = FormatsCustomFormatDialect;

/**
 * A condition's value as the editor holds it, by the shape of its kind's
 * value: a pattern, one of the kind's values, or a range of numbers.
 * `picked` is a value as text, as a Select holds it.
 */
export type Value =
  | { readonly shape: "pattern"; readonly text: string }
  | { readonly shape: "pick"; readonly picked: string; readonly except: boolean }
  | { readonly shape: "range"; readonly min: string; readonly max: string };

export type ConditionDraft = {
  /** Keeps a row's state while rows above it are removed. */
  readonly key: string;
  readonly kind: Kind;
  readonly name: string;
  readonly negate: boolean;
  readonly required: boolean;
  readonly value: Value;
};

export type FormatDraft = {
  readonly name: string;
  readonly includeWhenRenaming: boolean;
  readonly dialect: Dialect;
  readonly conditions: readonly ConditionDraft[];
};

/** Problems by the field they sit under, keyed as the API's field errors are. */
export type Problems = Readonly<Record<string, string>>;

export function kindsIn(schema: FormatsSchema, dialect: Dialect): readonly Kind[] {
  const offered = schema.dialects.find((each) => each.dialect === dialect);
  if (offered === undefined) {
    throw new Error(`the schema has no ${dialect} kinds`);
  }
  return offered.kinds;
}

export const excepts = (kind: Kind): boolean => kind.fields.includes("exceptLanguage");

const shown = (raw: unknown): string =>
  typeof raw === "string" || typeof raw === "number" ? String(raw) : "";

const nextKey = (() => {
  let count = 0;
  return () => {
    count += 1;
    return `condition-${String(count)}`;
  };
})();

export function blankValue(kind: Kind): Value {
  if (kind.pattern) {
    return { shape: "pattern", text: "" };
  }
  const [first] = kind.values;
  if (first !== undefined) {
    return { shape: "pick", picked: String(first.value), except: false };
  }
  return { shape: "range", min: "", max: "" };
}

export function blankCondition(kind: Kind): ConditionDraft {
  return {
    key: nextKey(),
    kind,
    name: "",
    negate: false,
    required: false,
    value: blankValue(kind),
  };
}

function valueOf(kind: Kind, fields: FormatsCondition["fields"]): Value {
  const blank = blankValue(kind);
  switch (blank.shape) {
    case "pattern":
      return { shape: "pattern", text: shown(fields["value"]) };
    case "pick":
      return {
        shape: "pick",
        picked: shown(fields["value"]),
        except: fields["exceptLanguage"] === true,
      };
    case "range":
      return { shape: "range", min: shown(fields["min"]), max: shown(fields["max"]) };
    default:
      return blank satisfies never;
  }
}

function kindNamed(kinds: readonly Kind[], name: string): Kind {
  const kind = kinds.find((offered) => offered.kind === name);
  if (kind === undefined) {
    throw new Error(`the schema has no kind ${name} for this format's app`);
  }
  return kind;
}

export function conditionsOf(
  format: Pick<FormatsCustomFormat, "conditions">,
  kinds: readonly Kind[],
): ConditionDraft[] {
  return format.conditions.map((condition) => {
    const kind = kindNamed(kinds, condition.kind);
    return {
      key: nextKey(),
      kind,
      name: condition.name,
      negate: condition.negate,
      required: condition.required,
      value: valueOf(kind, condition.fields),
    };
  });
}

export const draftOf = (format: FormatsCustomFormat, schema: FormatsSchema): FormatDraft => ({
  name: format.name,
  includeWhenRenaming: format.includeWhenRenaming,
  dialect: format.dialect,
  conditions: conditionsOf(format, kindsIn(schema, format.dialect)),
});

/** A new format starts with one empty condition of the first kind, which is Release title. */
export function newDraft(schema: FormatsSchema): FormatDraft {
  const [first] = kindsIn(schema, "slipstream");
  return {
    name: "",
    includeWhenRenaming: false,
    dialect: "slipstream",
    conditions: first === undefined ? [] : [blankCondition(first)],
  };
}

export const pickedWords = (kind: Kind, picked: string): string =>
  kind.values.find((value) => String(value.value) === picked)?.words ?? picked;

/** What a condition left unnamed is called: its value in words. */
export function valueWords(condition: ConditionDraft): string {
  const { value } = condition;
  switch (value.shape) {
    case "pattern":
      return value.text.trim();
    case "pick":
      return pickedWords(condition.kind, value.picked);
    case "range":
      return value.min.trim() === "" || value.max.trim() === ""
        ? ""
        : `${value.min.trim()} to ${value.max.trim()}`;
    default:
      return value satisfies never;
  }
}

function fieldsOf({ kind, value }: ConditionDraft): FormatsCondition["fields"] {
  switch (value.shape) {
    case "pattern":
      return { value: value.text };
    case "pick": {
      const picked = kind.values.find((offered) => String(offered.value) === value.picked)?.value;
      return excepts(kind) ? { value: picked, exceptLanguage: value.except } : { value: picked };
    }
    case "range":
      return { min: Number(value.min), max: Number(value.max) };
    default:
      return value satisfies never;
  }
}

export const bodyOf = (draft: FormatDraft): FormatsNewFormat => ({
  name: draft.name.trim(),
  includeWhenRenaming: draft.includeWhenRenaming,
  conditions: draft.conditions.map((condition) => ({
    name: condition.name.trim() === "" ? valueWords(condition) : condition.name.trim(),
    kind: condition.kind.kind,
    negate: condition.negate,
    required: condition.required,
    fields: fieldsOf(condition),
  })),
});

const isNumber = (text: string): boolean => text.trim() !== "" && Number.isFinite(Number(text));

function valueProblems(at: string, value: Value): Record<string, string> {
  switch (value.shape) {
    case "pattern":
      return value.text.trim() === "" ? { [`${at}.fields.value`]: "Enter a pattern." } : {};
    case "pick":
      return {};
    case "range":
      return {
        ...(!isNumber(value.min) && { [`${at}.fields.min`]: "Enter a number." }),
        ...(!isNumber(value.max) && { [`${at}.fields.max`]: "Enter a number." }),
      };
    default:
      return value satisfies never;
  }
}

function nameProblem(name: string): string | undefined {
  if (name === "") {
    return "Name the Custom Format.";
  }
  return name.length > 100 ? "Use 100 characters or fewer." : undefined;
}

export function problemsOf(draft: FormatDraft): Problems {
  const named = nameProblem(draft.name.trim());
  return {
    ...(named !== undefined && { name: named }),
    ...(draft.conditions.length === 0 && { conditions: "Add a condition." }),
    ...Object.fromEntries(
      draft.conditions.flatMap((condition, at) =>
        Object.entries(valueProblems(`conditions[${String(at)}]`, condition.value)),
      ),
    ),
  };
}
