import { MoreHorizontalIcon, Search01Icon } from "@hugeicons/core-free-icons";

import { IconButton } from "../../ui/icon-button";
import { Menu, type MenuItem } from "../../ui/menu";
import { RowSwitch } from "../../ui/row-switch";
import type { RowActions } from "./item-row";
import type { HeaderActions } from "./level-header";
import type { LevelSearch } from "./level-search";
import type { ItemView, LevelView } from "./model";

function MoreMenu({ name, items }: { name: string; items: readonly MenuItem[] }) {
  return (
    <Menu
      label={`Actions for ${name}`}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`More actions for ${name}`} />}
      items={items}
    />
  );
}

const iconPlace = <span aria-hidden="true" className="size-(--icon-button-size) shrink-0" />;

/** Manual search, and Grab now while a cell of the level waits for a better Release. */
function levelMenu(
  level: LevelView,
  search: LevelSearch | undefined,
  actions: HeaderActions,
): MenuItem[] {
  const manual: MenuItem = {
    label: "Manual search",
    onSelect: () => {
      actions.manual(level);
    },
  };
  if (search === undefined || search.waits.length === 0) {
    return [manual];
  }
  return [
    manual,
    {
      label: "Grab now",
      onSelect: () => {
        actions.grabNow(level, search);
      },
    },
  ];
}

/** The withdrawn section has no switch, Search or menu, only their places. */
export function LevelControls({
  level,
  phone,
  search,
  actions,
}: {
  level: LevelView;
  phone: boolean;
  search: LevelSearch | undefined;
  actions: HeaderActions;
}) {
  if (level.withdrawn) {
    return (
      <>
        <span className="w-8.5 shrink-0" />
        {iconPlace}
        {!phone && iconPlace}
      </>
    );
  }
  return (
    <>
      <RowSwitch
        label={`Monitor ${level.name}`}
        checked={level.monitor}
        onCheckedChange={(on) => {
          actions.monitor(level, on);
        }}
      />
      <IconButton
        icon={Search01Icon}
        label={`Search ${level.name}`}
        busy={search?.searching ?? false}
        onClick={() => {
          actions.search(level);
        }}
      />
      <MoreMenu name={level.name} items={levelMenu(level, search, actions)} />
    </>
  );
}

/** A withdrawn Item has no Monitor switch or Search, only their places. */
export function RowControls({ view, actions }: { view: ItemView; actions: RowActions }) {
  if (view.withdrawn) {
    return (
      <>
        <span className="w-8.5 shrink-0" />
        <span aria-hidden="true" className="size-(--icon-button-size) shrink-0" />
      </>
    );
  }
  return (
    <>
      <span className="flex w-8.5 shrink-0">
        <RowSwitch
          label={`Monitor ${view.code}`}
          checked={view.item.monitored}
          onCheckedChange={(on) => {
            actions.monitor(view, on);
          }}
        />
      </span>
      <IconButton
        icon={Search01Icon}
        label={`Search ${view.code}`}
        onClick={() => {
          actions.search(view);
        }}
      />
    </>
  );
}

/** Details, and Manual search unless the Item's source withdrew it. */
export function RowMenu({ view, actions }: { view: ItemView; actions: RowActions }) {
  const details: MenuItem = {
    label: "Details",
    onSelect: () => {
      actions.details(view);
    },
  };
  const manual: MenuItem = {
    label: "Manual search",
    onSelect: () => {
      actions.manual(view);
    },
  };
  return <MoreMenu name={view.code} items={view.withdrawn ? [details] : [details, manual]} />;
}
