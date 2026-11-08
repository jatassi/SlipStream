import { DraftField } from "../targets/field";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";
import { type Kind, type Problems, type Value, excepts } from "./draft";

type Shaped<S extends Value["shape"]> = {
  readonly kind: Kind;
  readonly value: Extract<Value, { shape: S }>;
  readonly at: string;
  readonly problems: Problems;
  readonly onValue: (value: Value) => void;
};

function PatternField({ value, at, problems, onValue }: Shaped<"pattern">) {
  return (
    <DraftField
      label="Pattern"
      value={value.text}
      description="A .NET regular expression, matched ignoring case."
      error={problems[`${at}.fields.value`]}
      onChange={(text) => {
        onValue({ ...value, text });
      }}
    />
  );
}

function PickField({ kind, value, onValue }: Shaped<"pick">) {
  return (
    <div className="flex flex-col gap-1">
      <Select
        label={kind.words}
        options={kind.values.map((offered) => ({
          value: String(offered.value),
          label: offered.words,
        }))}
        value={value.picked}
        onValueChange={(picked) => {
          onValue({ ...value, picked });
        }}
      />
      {excepts(kind) && (
        <Checkbox
          label="Any language but this one"
          checked={value.except}
          onCheckedChange={(except) => {
            onValue({ ...value, except });
          }}
        />
      )}
    </div>
  );
}

function RangeField({ kind, value, at, problems, onValue }: Shaped<"range">) {
  const [low, high] =
    kind.kind === "SizeSpecification" ? ["Over (GB)", "At most (GB)"] : ["From", "To"];
  return (
    <div className="grid grid-cols-2 gap-3">
      <DraftField
        label={low}
        value={value.min}
        inputMode="decimal"
        error={problems[`${at}.fields.min`]}
        onChange={(min) => {
          onValue({ ...value, min });
        }}
      />
      <DraftField
        label={high}
        value={value.max}
        inputMode="decimal"
        error={problems[`${at}.fields.max`]}
        onChange={(max) => {
          onValue({ ...value, max });
        }}
      />
    </div>
  );
}

export function ValueField({
  kind,
  value,
  at,
  problems,
  onValue,
}: {
  kind: Kind;
  value: Value;
  at: string;
  problems: Problems;
  onValue: (value: Value) => void;
}) {
  const common = { kind, at, problems, onValue };
  switch (value.shape) {
    case "pattern":
      return <PatternField value={value} {...common} />;
    case "pick":
      return <PickField value={value} {...common} />;
    case "range":
      return <RangeField value={value} {...common} />;
    default:
      return value satisfies never;
  }
}
