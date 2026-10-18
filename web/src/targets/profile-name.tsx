import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Select } from "../ui/select";
import type { SectionProps } from "./editor";
import { DraftField } from "./field";
import { type ProfileDraft, newProfileDraft } from "./profile-draft";

function ContentType({ draft, onChange }: SectionProps<ProfileDraft>) {
  const { contentTypes } = useListContentTypesSuspense().data;
  const type = contentTypes.find((candidate) => candidate.id === draft.contentType);
  if (draft.id !== null) {
    return (
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        For {type?.work.many ?? draft.contentType}. A Profile&apos;s Content Type is fixed.
      </p>
    );
  }
  return (
    <Select
      label="Content Type"
      options={contentTypes.map((candidate) => ({
        value: candidate.id,
        label: candidate.work.many,
      }))}
      value={draft.contentType}
      onValueChange={(contentType) => {
        onChange({ ...draft, contentType, editions: newProfileDraft(contentType).editions });
      }}
    />
  );
}

export function ProfileNameSection({ draft, onChange }: SectionProps<ProfileDraft>) {
  return (
    <div className="flex max-w-[400px] flex-col gap-4">
      <DraftField
        label="Name"
        value={draft.name}
        description="Shown where a Work picks its Profile."
        onChange={(name) => {
          onChange({ ...draft, name });
        }}
      />
      <ContentType draft={draft} onChange={onChange} />
    </div>
  );
}
