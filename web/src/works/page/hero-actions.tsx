import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import type { ReactElement, ReactNode } from "react";

import type { WorksWorkView } from "../../api/gen/model";
import { SearchAction } from "../../grabs/search";
import { Button } from "../../ui/button";
import { IconButton } from "../../ui/icon-button";
import { Menu, type MenuEntry } from "../../ui/menu";
import { useShell } from "../../ui/shell-context";
import { Switch } from "../../ui/switch";

export type HeroActions = {
  /** The Editions control on its trigger, or its sheet alone with none, as on phone. */
  readonly editions: (trigger: ReactElement | undefined) => ReactNode;
  /** Where the Work's Edition list comes from, beside the Editions button. */
  readonly editionsFrom: string;
  readonly openEditions: () => void;
  readonly monitored: {
    readonly on: boolean;
    readonly busy: boolean;
    readonly set: (on: boolean) => void;
  };
  readonly menu: readonly MenuEntry[];
};

// On phone the actions past Search sit in one action sheet, with Editions as "Edit Editions".
export function Actions({ work, actions }: { work: WorksWorkView; actions: HeroActions }) {
  const more = <IconButton icon={MoreHorizontalIcon} label={`More actions for ${work.label}`} />;
  if (useShell() === "phone") {
    return (
      <div className="flex items-center gap-2">
        <SearchAction workId={work.id} />
        <Menu
          label={`Actions for ${work.label}`}
          trigger={more}
          items={[
            {
              kind: "toggle",
              label: "Monitored",
              checked: actions.monitored.on,
              onCheckedChange: actions.monitored.set,
            },
            { label: "Edit Editions", onSelect: actions.openEditions },
            "separator",
            ...actions.menu,
          ]}
        />
        {actions.editions(undefined)}
      </div>
    );
  }
  return (
    <div className="flex flex-wrap items-center gap-3">
      <SearchAction workId={work.id} />
      {actions.editions(
        <Button>
          Editions <span className="font-normal text-ink-3">{actions.editionsFrom}</span>
        </Button>,
      )}
      <Switch
        label="Monitored"
        checked={actions.monitored.on}
        disabled={actions.monitored.busy}
        onCheckedChange={actions.monitored.set}
      />
      <Menu label={`Actions for ${work.label}`} trigger={more} items={actions.menu} />
    </div>
  );
}
