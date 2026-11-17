import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
import { type ReactNode, useId } from "react";

/** A description sits under its label and is read as the radio's description. */
export type RadioOption<V extends string> = { value: V; label: string; description?: string };

export function RadioGroup<V extends string>({
  label,
  options,
  value,
  onValueChange,
}: {
  label: string;
  options: readonly RadioOption<V>[];
  value: V;
  onValueChange: (value: V) => void;
}) {
  const id = useId();
  return (
    <BaseRadioGroup
      aria-label={label}
      value={value}
      onValueChange={(next) => {
        onValueChange(next);
      }}
      className="flex flex-col"
    >
      {options.map((option) => {
        const noteId = `${id}-${option.value}`;
        return (
          <div key={option.value} className="flex flex-col">
            <label className="inline-flex min-h-(--control-height) cursor-default items-center gap-2.5 text-body text-ink-1 select-none in-data-[density=touch]:gap-3 in-data-[density=touch]:text-body-touch">
              <Radio.Root
                value={option.value}
                aria-describedby={option.description === undefined ? undefined : noteId}
                className="flex size-4 shrink-0 items-center justify-center rounded-capsule bg-well ring-1 ring-hairline-strong ring-inset transition-[background-color] duration-(--duration-control) ease-standard in-data-[density=touch]:size-5.5 data-checked:bg-[hsl(var(--h)_62%_46%)] data-checked:ring-0 motion-reduce:transition-none"
              >
                <Radio.Indicator className="size-1.5 rounded-capsule bg-solid in-data-[density=touch]:size-2" />
              </Radio.Root>
              {option.label}
            </label>
            {option.description !== undefined && (
              <p
                id={noteId}
                className="-mt-1 mb-1 pl-6.5 text-meta text-ink-3 in-data-[density=touch]:pl-8.5 in-data-[density=touch]:text-body-touch"
              >
                {option.description}
              </p>
            )}
          </div>
        );
      })}
    </BaseRadioGroup>
  );
}

export type RadioCard<V extends string> = { value: V; label: string; mark: ReactNode };

const card =
  "flex min-h-14 items-center gap-3 rounded-card bg-glass px-3 py-2.5 text-left text-body font-medium text-ink-1 ring-1 ring-hairline ring-inset transition-[background-color,box-shadow] duration-(--duration-control) ease-standard select-none hover:bg-fill in-data-[density=touch]:text-body-touch data-checked:bg-selected data-checked:ring-2 data-checked:ring-glow motion-reduce:transition-none";

/** Radio cards: each option its mark and name, three columns from a 448px container and two below. */
export function RadioCards<V extends string>({
  label,
  options,
  value,
  onValueChange,
}: {
  label: string;
  options: readonly RadioCard<V>[];
  value: V;
  onValueChange: (value: V) => void;
}) {
  return (
    <div className="@container">
      <BaseRadioGroup
        aria-label={label}
        value={value}
        onValueChange={(next) => {
          onValueChange(next);
        }}
        className="grid grid-cols-2 gap-2 @md:grid-cols-3"
      >
        {options.map((option) => (
          <Radio.Root key={option.value} value={option.value} className={card}>
            {option.mark}
            <span className="min-w-0">{option.label}</span>
          </Radio.Root>
        ))}
      </BaseRadioGroup>
    </div>
  );
}
