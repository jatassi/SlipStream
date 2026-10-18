import type { SectionProps } from "./editor";
import { EditionsControl } from "./editions-control";
import type { ProfileDraft } from "./profile-draft";

export function ProfileEditionsSection({ draft, onChange }: SectionProps<ProfileDraft>) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">
        The cuts this Profile&apos;s Works want. A Work can set its own list in place of this one.
      </p>
      <EditionsControl
        value={draft.editions}
        contentType={draft.contentType}
        onChange={(editions) => {
          onChange({ ...draft, editions });
        }}
      />
    </div>
  );
}
