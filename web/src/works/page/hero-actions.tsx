import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import type { ReactElement, ReactNode } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
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

const monitoringEntry = (monitoring: HeroActions["monitoring"]): MenuEntry =>
  monitoring.kind === "switch"
    ? {
        kind: "toggle",
        label: "Monitored",
        checked: monitoring.on,
        onCheckedChange: monitoring.set,
      }
    : { label: "Set monitoring…", onSelect: monitoring.open };

// On phone the actions past Search sit in one action sheet, with Editions as "Edit Editions".
export function Actions({
  type,
  work,
  actions,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  actions: HeroActions;
}) {
  const more = <IconButton icon={MoreHorizontalIcon} label={`More actions for ${work.label}`} />;
  if (useShell() === "phone") {
    return (
      <div className="flex items-center gap-2">
        <SearchAction workId={work.id} type={type} />
        <Menu
          label={`Actions for ${work.label}`}
          trigger={more}
          items={[
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
  return (
    <div className="flex flex-wrap items-center gap-3">
      <SearchAction workId={work.id} type={type} />
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
