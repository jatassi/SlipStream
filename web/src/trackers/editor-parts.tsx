import type { ReactNode } from "react";

import { SubmitButton } from "../ui/form";
import { FormAlert } from "../ui/form-alert";
import { Segmented, type SegmentedOption } from "../ui/segmented";

/** A seeding editor's form, which saves its whole value. */
export function EditorForm({
  label,
  submit,
  children,
}: {
  label: string;
  submit: () => Promise<void>;
  children: ReactNode;
}) {
  return (
    <form
      noValidate
      aria-label={label}
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      {children}
    </form>
  );
}

/** A labelled choice between a few ways a rule or goal is met. */
export function Choice<V extends string>({
  label,
  options,
  value,
  onValueChange,
}: {
  label: string;
  options: readonly SegmentedOption<V>[];
  value: V;
  onValueChange: (value: V) => void;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span
        aria-hidden="true"
        className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch"
      >
        {label}
      </span>
      <Segmented label={label} options={options} value={value} onValueChange={onValueChange} />
    </div>
  );
}

/** The form's refusal, Save, and an action beside it such as "Use the default goal". */
export function EditorActions({ save, aside }: { save: string; aside: ReactNode }) {
  return (
    <>
      <FormAlert />
      <div className="flex flex-wrap items-center gap-2">
        <SubmitButton label={save} />
        {aside}
      </div>
    </>
  );
}
