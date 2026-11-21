import { useFormContext } from "./form-context";
import { SignalLine } from "./signal-line";

export function FormAlert() {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state): unknown => state.errorMap.onSubmit}>
      {(error) =>
        typeof error === "string" ? (
          <SignalLine tone="fail" announce>
            {error}
          </SignalLine>
        ) : null
      }
    </form.Subscribe>
  );
}
