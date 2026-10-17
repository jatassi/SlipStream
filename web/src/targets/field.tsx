import { Alert02Icon } from "@hugeicons/core-free-icons";
import { useId } from "react";

import { Icon } from "../ui/icon";

const labelLook =
  "text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold";

const inputLook =
  "h-(--input-height) w-full min-w-0 rounded-item bg-well px-3 text-body text-ink-1 ring-1 ring-hairline ring-inset placeholder:text-ink-3 aria-invalid:ring-fail-line in-data-[density=touch]:text-input-touch";

/** A labelled text input over a draft's value, styled as the kit's text field. */
export function DraftField({
  label,
  value,
  onChange,
  description,
  error,
  placeholder,
  inputMode,
  name,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  description?: string | undefined;
  error?: string | undefined;
  placeholder?: string;
  inputMode?: "decimal";
  /** The input's accessible name, when the label alone repeats on every row. */
  name?: string;
}) {
  const id = useId();
  const noteId = useId();
  const described = error !== undefined || description !== undefined;
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <label htmlFor={id} className={labelLook}>
        {label}
      </label>
      <input
        id={id}
        value={value}
        placeholder={placeholder}
        inputMode={inputMode}
        aria-label={name}
        aria-invalid={error === undefined ? undefined : true}
        aria-describedby={described ? noteId : undefined}
        onChange={(event) => {
          onChange(event.target.value);
        }}
        className={inputLook}
      />
      <Note id={noteId} error={error} description={description} />
    </div>
  );
}

function Note({
  id,
  error,
  description,
}: {
  id: string;
  error: string | undefined;
  description: string | undefined;
}) {
  if (error !== undefined) {
    return (
      <p
        id={id}
        className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
      >
        <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
        {error}
      </p>
    );
  }
  return description === undefined ? null : (
    <p id={id} className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
      {description}
    </p>
  );
}
