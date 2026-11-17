import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { ClearDialog } from "./clear-dialog";
import { KeepDialog } from "./keep-dialog";

/** History options: how long History keeps entries, and Clear History with its confirmation. */
export function HistoryMenu() {
  const [open, setOpen] = useState<"keep" | "clear">();
  const close = () => {
    setOpen(undefined);
  };
  return (
    <>
      <Menu
        label="History options"
        trigger={<IconButton icon={MoreHorizontalIcon} label="History options" />}
        items={[
          {
            label: "Keep history for…",
            onSelect: () => {
              setOpen("keep");
            },
          },
          {
            label: "Clear History…",
            tone: "danger",
            onSelect: () => {
              setOpen("clear");
            },
          },
        ]}
      />
      <KeepDialog open={open === "keep"} onClose={close} />
      <ClearDialog open={open === "clear"} onClose={close} />
    </>
  );
}
