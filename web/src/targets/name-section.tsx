import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";
import { DraftField } from "./field";

export function NameSection({ draft, onChange }: SectionProps<TargetDraft>) {
  return (
    <div className="max-w-[400px]">
      <DraftField
        label="Name"
        value={draft.name}
        description="Your Media Server's version picker shows this name."
        onChange={(name) => {
          onChange({ ...draft, name });
        }}
      />
    </div>
  );
}
