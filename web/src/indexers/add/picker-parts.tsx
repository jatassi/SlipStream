import { ArrowRight01Icon } from "@hugeicons/core-free-icons";

import type { IndexersCatalogEntry } from "../../api/gen/model";
import { Chip } from "../../ui/chip";
import { Icon } from "../../ui/icon";
import { Segmented } from "../../ui/segmented";
import { Select } from "../../ui/select";
import { mediaWords } from "./families";
import {
  type Filter,
  type Privacy,
  anyLanguage,
  languageName,
  privacyWords,
} from "./picker-filter";

export const unrunnable = "SlipStream can't run this definition yet";
export const replaced = "Replaced by your custom definition";

/** A catalog entry's second line: why it can't be added, or what it is. */
function metaOf(entry: IndexersCatalogEntry): string {
  if (entry.replacedByCustom) {
    return replaced;
  }
  if (entry.reason !== null) {
    return unrunnable;
  }
  const media = mediaWords(entry.categories);
  const parts = [
    entry.origin === "custom" ? "Custom definition" : undefined,
    media === "" ? undefined : media,
    languageName(entry.language),
  ];
  return parts.filter((part) => part !== undefined).join(" · ");
}

const greyed = (entry: IndexersCatalogEntry) => entry.reason !== null || entry.replacedByCustom;

export function EntryRow({ entry, active }: { entry: IndexersCatalogEntry; active: boolean }) {
  return (
    <div
      className={`flex min-h-(--row-height-two) cursor-default items-center gap-3 px-4 py-2 shadow-[inset_0_1px_0_var(--color-hairline)] transition-colors duration-350 ease-standard in-[[aria-posinset='1']]:shadow-none hover:bg-(--row-wash) ${active ? "bg-(--row-wash)" : ""}`}
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className={`truncate text-(length:--body-size) in-data-[density=touch]:text-card-title ${greyed(entry) ? "text-ink-3" : "text-ink-1"}`}
        >
          {entry.name}
        </span>
        <span className="truncate text-meta text-ink-3">{metaOf(entry)}</span>
      </span>
      <Chip>{privacyWords[entry.privacy]}</Chip>
      <Icon icon={ArrowRight01Icon} className="size-4 shrink-0 text-ink-3" />
    </div>
  );
}

export function Filters({
  filter,
  counts,
  languages,
  onChange,
}: {
  filter: Filter;
  counts: Record<Privacy, number>;
  languages: { value: string; label: string }[];
  onChange: (next: Filter) => void;
}) {
  const privacies = [
    { value: "all", label: "All", count: counts.all },
    { value: "private", label: privacyWords.private, count: counts.private },
    { value: "semi-private", label: privacyWords["semi-private"], count: counts["semi-private"] },
    { value: "public", label: privacyWords.public, count: counts.public },
  ] as const;
  return (
    <div className="flex flex-wrap items-end gap-3">
      <Segmented
        label="Privacy"
        options={privacies}
        value={filter.privacy}
        onValueChange={(privacy) => {
          onChange({ ...filter, privacy });
        }}
      />
      <div className="w-56">
        <Select
          label="Language"
          options={[{ value: anyLanguage, label: "Any language" }, ...languages]}
          value={filter.language}
          onValueChange={(language) => {
            onChange({ ...filter, language });
          }}
        />
      </div>
    </div>
  );
}
