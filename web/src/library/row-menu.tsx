import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import { useStartSearch } from "../grabs/search";
import { openManualSearch } from "../manual/sheet-state";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";

/** A Library Work's menu: Search, its gone files included, and Manual search. */
export function WorkMenu({
  work,
  className = "",
}: {
  work: { readonly id: string; readonly label: string };
  className?: string;
}) {
  const search = useStartSearch();
  return (
    <Menu
      label={`Actions for ${work.label}`}
      trigger={
        <IconButton
          icon={MoreHorizontalIcon}
          label={`More actions for ${work.label}`}
          className={className}
        />
      }
      items={[
        {
          label: "Search",
          onSelect: () => {
            search.start(work.id, { name: work.label });
          },
        },
        {
          label: "Manual search",
          onSelect: () => {
            openManualSearch({ workId: work.id });
          },
        },
      ]}
    />
  );
}
