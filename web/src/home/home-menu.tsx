import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useState } from "react";

import {
  getGetHomeSettingsSuspenseQueryOptions,
  updateHomeSettings,
} from "../api/gen/query/home/home";
import { adoptHomeSettings } from "../live/adopt-home-settings";
import { Dialog } from "../ui/dialog";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { SettingField } from "../ui/setting-field";

const rowsKey = "home.rows";

function RowsField() {
  const queryClient = useQueryClient();
  const { settings } = useSuspenseQuery(getGetHomeSettingsSuspenseQueryOptions()).data;
  const rows = settings.find((field) => field.key === rowsKey);
  if (rows?.kind !== "count") {
    throw new Error("Home has no Rows on Home");
  }
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-2">How many entries Needs you, Jobs and Recent each show.</p>
      <SettingField
        field={rows}
        onSave={async (value) => {
          await adoptHomeSettings(
            queryClient,
            await updateHomeSettings({ values: { [rowsKey]: value } }),
          );
        }}
      />
    </div>
  );
}

/** Home options: Rows on Home, set where its effect shows. */
export function HomeMenu() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Menu
        label="Home options"
        trigger={<IconButton icon={MoreHorizontalIcon} label="Home options" />}
        items={[
          {
            label: "Rows on Home…",
            onSelect: () => {
              setOpen(true);
            },
          },
        ]}
      />
      <Dialog title="Home" open={open} onOpenChange={setOpen}>
        {open && <RowsField />}
      </Dialog>
    </>
  );
}
