import { Checkbox } from "../../ui/checkbox";
import { errorText } from "../../ui/form-context";
import { familiesOf, familyName } from "./families";
import type { AddForm } from "./use-add";

/** Checkboxes over a set of choices, saved as the chosen values. */
export function ChoiceSet({
  legend,
  options,
  chosen,
  errors,
  description,
  onChange,
}: {
  legend: string;
  options: readonly { value: string; label: string }[];
  chosen: readonly string[];
  errors: readonly unknown[];
  description?: string | undefined;
  onChange: (chosen: string[]) => void;
}) {
  const error = errorText(errors);
  return (
    <fieldset className="flex flex-col gap-1">
      <legend className="mb-1 font-semibold text-ink-2">{legend}</legend>
      {options.map((option) => (
        <Checkbox
          key={option.value}
          label={option.label}
          checked={chosen.includes(option.value)}
          onCheckedChange={(checked) => {
            onChange(
              checked ? [...chosen, option.value] : chosen.filter((kept) => kept !== option.value),
            );
          }}
        />
      ))}
      {error === undefined ? (
        description !== undefined && (
          <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            {description}
          </p>
        )
      ) : (
        <p role="alert" className="text-meta text-fail-ink">
          {error}
        </p>
      )}
    </fieldset>
  );
}

/** The kinds of media a definition's categories fall in, narrowable to fewer. */
export function Families({ form, categories }: { form: AddForm; categories: readonly number[] }) {
  const offered = familiesOf(categories);
  if (offered.length < 2) {
    return null;
  }
  return (
    <form.Field
      name="families"
      validators={{
        onSubmit: ({ value }) =>
          value.length === 0 ? "Keep at least one kind of media." : undefined,
      }}
    >
      {(field) => (
        <ChoiceSet
          legend="Categories"
          options={offered.map((family) => ({ value: String(family), label: familyName(family) }))}
          chosen={field.state.value}
          errors={field.state.meta.errors}
          description="Searches ask only for the kinds of media kept here."
          onChange={field.handleChange}
        />
      )}
    </form.Field>
  );
}
