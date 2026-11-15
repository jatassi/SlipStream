import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import type { ReactElement, ReactNode } from "react";

import type { WorksWorkView } from "../../api/gen/model";
import { ManualSearchButton, SearchAction, type Searched } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
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
  /** A Work of one Item has a Monitored switch; one of many opens the Monitoring dialog. */
  readonly monitoring:
    | {
        readonly kind: "switch";
        readonly on: boolean;
        readonly busy: boolean;
        readonly set: (on: boolean) => void;
      }
    | { readonly kind: "dialog"; readonly words: string; readonly open: () => void };
  readonly menu: readonly MenuEntry[];
};

/** The hero's actions with what its Search names in the toast. */
export type HeroControls = HeroActions & { readonly searched: Searched };

const monitoringEntry = (monitoring: HeroActions["monitoring"]): MenuEntry =>
  monitoring.kind === "switch"
    ? {
        kind: "toggle",
        label: "Monitored",
        checked: monitoring.on,
        onCheckedChange: monitoring.set,
      }
    : { label: "Set monitoring…", onSelect: monitoring.open };

// On phone the actions past Search sit in one action sheet, Manual search first and Editions as "Edit Editions".
function PhoneActions({
  work,
  actions,
  more,
}: {
  work: WorksWorkView;
  actions: HeroControls;
  more: ReactElement;
}) {
  return (
    <div className="flex items-center gap-2">
      <SearchAction workId={work.id} searched={actions.searched} />
      <Menu
        label={`Actions for ${work.label}`}
        trigger={more}
        items={[
          {
            label: "Manual search",
            onSelect: () => {
              openManualSearch({ workId: work.id });
            },
          },
          "separator",
          monitoringEntry(actions.monitoring),
          { label: "Edit Editions", onSelect: actions.openEditions },
          "separator",
          ...actions.menu,
        ]}
      />
      {actions.editions(undefined)}
    </div>
  );
}

export function Actions({ work, actions }: { work: WorksWorkView; actions: HeroControls }) {
  const more = <IconButton icon={MoreHorizontalIcon} label={`More actions for ${work.label}`} />;
  if (useShell() === "phone") {
    return <PhoneActions work={work} actions={actions} more={more} />;
  }
  return (
    <div className="flex flex-wrap items-center gap-3">
      <SearchAction workId={work.id} searched={actions.searched} />
      <ManualSearchButton workId={work.id} />
      {actions.editions(
        <Button>
          Editions <span className="font-normal text-ink-3">{actions.editionsFrom}</span>
        </Button>,
      )}
      {actions.monitoring.kind === "switch" ? (
        <Switch
          label="Monitored"
          checked={actions.monitoring.on}
          disabled={actions.monitoring.busy}
          onCheckedChange={actions.monitoring.set}
        />
      ) : (
        <Button onClick={actions.monitoring.open}>{actions.monitoring.words}</Button>
      )}
      <Menu label={`Actions for ${work.label}`} trigger={more} items={actions.menu} />
    </div>
  );
}
