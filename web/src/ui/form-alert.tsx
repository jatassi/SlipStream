import { useFormContext } from "./form-context";

export function FormAlert() {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state): unknown => state.errorMap.onSubmit}>
      {(error) =>
        typeof error === "string" ? (
          <p role="alert" className="text-fail-ink">
            {error}
          </p>
        ) : null
      }
    </form.Subscribe>
  );
}
