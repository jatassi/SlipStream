import { useRef, useState } from "react";

import { errorText, useFieldContext } from "./form-context";
import { PinInput, type PinInputProps, pinLength } from "./pin-field";
import { usePortalDensity } from "./portal-density";

const digits = ["1", "2", "3", "4", "5", "6", "7", "8", "9"] as const;

const key =
  "grid size-20 place-items-center rounded-capsule bg-fill text-numeral font-normal text-ink-1 ring-1 ring-hairline ring-inset transition-colors duration-250 ease-standard select-none active:bg-selected active:duration-0 disabled:opacity-40 motion-reduce:transition-none";

function Keypad({
  value,
  onValueChange,
  onComplete,
}: {
  value: string;
  onValueChange: (value: string) => void;
  onComplete: (value: string) => void;
}) {
  const press = (digit: string) => {
    if (value.length >= pinLength) {
      return;
    }
    const next = value + digit;
    onValueChange(next);
    if (next.length === pinLength) {
      onComplete(next);
    }
  };
  const digitKey = (digit: string) => (
    <button
      key={digit}
      type="button"
      className={key}
      onClick={() => {
        press(digit);
      }}
    >
      {digit}
    </button>
  );
  return (
    <fieldset aria-label="Keypad" className="grid grid-cols-3 gap-x-6 gap-y-4">
      {digits.map((digit) => digitKey(digit))}
      <span aria-hidden="true" />
      {digitKey("0")}
      <button
        type="button"
        disabled={value === ""}
        className="grid h-20 place-items-center rounded-capsule text-body-touch font-semibold text-ink-1 select-none disabled:opacity-40"
        onClick={() => {
          onValueChange(value.slice(0, -1));
        }}
      >
        Delete
      </button>
    </fieldset>
  );
}

type EntryProps = Omit<PinInputProps, "keypad" | "autoSubmit" | "onComplete"> & {
  onComplete: (value: string) => void;
  shakes?: number;
};

// At the Touch density the keypad takes the PIN and the field above only shows
// it, so the system keyboard stays down. A hardware keyboard still types there.
function PinEntry({ value, onValueChange, onComplete, shakes = 0, ...input }: EntryProps) {
  const { density, probe } = usePortalDensity();
  const keypad = density === "touch";
  return (
    <div className="flex flex-col gap-6 in-data-[density=touch]:items-center">
      {probe}
      <PinInput
        key={shakes}
        {...input}
        value={value}
        onValueChange={onValueChange}
        onComplete={onComplete}
        keypad={keypad}
        className={`in-data-[density=touch]:items-center ${shakes > 0 ? "motion-safe:animate-shake" : ""}`}
      />
      {keypad && <Keypad value={value} onValueChange={onValueChange} onComplete={onComplete} />}
    </div>
  );
}

function useSubmit() {
  const ref = useRef<HTMLDivElement>(null);
  return { ref, submit: () => ref.current?.closest("form")?.requestSubmit() };
}

export function PinPad({
  label,
  autoComplete,
  autoSubmit = false,
}: {
  label: string;
  autoComplete: "current-password" | "new-password";
  autoSubmit?: boolean;
}) {
  const field = useFieldContext<string>();
  const { ref, submit } = useSubmit();
  return (
    <div ref={ref}>
      <PinEntry
        label={label}
        name={field.name}
        value={field.state.value}
        onValueChange={(value) => {
          field.handleChange(value);
        }}
        onComplete={() => {
          if (autoSubmit) {
            submit();
          }
        }}
        onBlur={field.handleBlur}
        autoComplete={autoComplete}
        error={errorText(field.state.meta.errors)}
      />
    </div>
  );
}

const mismatch = "Those PINs don't match. Choose a new PIN.";

/**
 * A new PIN, entered twice on the keypad. The field holds the PIN once both
 * entries match, and the form submits then.
 */
export function NewPinPad({ label = "New PIN" }: { label?: string }) {
  const field = useFieldContext<string>();
  const { ref, submit } = useSubmit();
  const [first, setFirst] = useState<string | null>(null);
  const [entry, setEntry] = useState("");
  const [shakes, setShakes] = useState(0);
  const [mismatched, setMismatched] = useState(false);
  const complete = (pin: string) => {
    setEntry("");
    if (first === null) {
      setFirst(pin);
      setMismatched(false);
      return;
    }
    setFirst(null);
    if (pin !== first) {
      setMismatched(true);
      setShakes((count) => count + 1);
      return;
    }
    field.handleChange(pin);
    submit();
  };
  return (
    <div ref={ref}>
      <PinEntry
        label={first === null ? label : "Type it again"}
        value={entry}
        onValueChange={(value) => {
          setEntry(value);
          if (value !== "") {
            field.handleChange("");
          }
        }}
        onComplete={complete}
        autoComplete="new-password"
        shakes={shakes}
        error={mismatched ? mismatch : errorText(field.state.meta.errors)}
      />
    </div>
  );
}
