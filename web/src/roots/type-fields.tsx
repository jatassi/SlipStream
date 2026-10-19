import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";

export function TypeSelect({
  type,
  onTypeChange,
}: {
  type: string;
  onTypeChange: (type: string) => void;
}) {
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <Select
      label="Content Type"
      options={contentTypes
        .filter((candidate) => candidate.enabled || candidate.id === type)
        .map((candidate) => ({ value: candidate.id, label: candidate.name }))}
      value={type}
      onValueChange={onTypeChange}
    />
  );
}

/** A type's first root is its default without asking; a later one may take over. */
export function DefaultChoice({
  type,
  makeDefault,
  onMakeDefaultChange,
}: {
  type: string;
  makeDefault: boolean;
  onMakeDefaultChange: (makeDefault: boolean) => void;
}) {
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const label = contentTypes.find((candidate) => candidate.id === type)?.work.many ?? type;
  return roots.some((root) => root.contentType === type) ? (
    <Checkbox
      label={`Make this the default for ${label}`}
      checked={makeDefault}
      onCheckedChange={onMakeDefaultChange}
    />
  ) : (
    <p className="text-ink-3">
      It becomes the default for {label}, where new {label} go.
    </p>
  );
}
