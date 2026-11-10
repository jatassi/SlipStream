import { Checkbox } from "../ui/checkbox";
import { SearchField } from "../ui/search-field";
import type { usePicks } from "./picks";

export function Toolbar({
  picks,
  draft,
  onDraft,
}: {
  picks: ReturnType<typeof usePicks> | undefined;
  draft: string;
  onDraft: (next: string) => void;
}) {
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {picks !== undefined && (
        <Checkbox
          label="Pick all"
          checked={picks.all === "every"}
          indeterminate={picks.all === "some"}
          onCheckedChange={picks.toggleAll}
        />
      )}
      <SearchField
        label="Search the Blocklist"
        placeholder="Search by title"
        value={draft}
        onValueChange={onDraft}
        className="min-w-0 flex-1 md:ml-auto md:w-72 md:flex-none"
      />
    </div>
  );
}
