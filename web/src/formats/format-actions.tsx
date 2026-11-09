import { Copy01Icon, Delete02Icon, MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { FormatsCustomFormat } from "../api/gen/model";
import { duplicateCustomFormat } from "../api/gen/query/formats/formats";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { toast } from "../ui/toast";
import { DeleteFormatDialog } from "./delete-format-dialog";

async function duplicating(format: FormatsCustomFormat) {
  try {
    const copy = await duplicateCustomFormat(format.id);
    toast(`Made ${copy.name} from ${format.name}.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export function FormatMenu({ format }: { format: FormatsCustomFormat }) {
  const [deleting, setDeleting] = useState(false);
  return (
    <>
      <Menu
        label={format.name}
        trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${format.name}`} />}
        items={[
          {
            label: "Duplicate",
            icon: Copy01Icon,
            onSelect: () => {
              void duplicating(format);
            },
          },
          "separator",
          {
            label: "Delete",
            icon: Delete02Icon,
            tone: "danger",
            onSelect: () => {
              setDeleting(true);
            },
          },
        ]}
      />
      <DeleteFormatDialog format={format} open={deleting} onOpenChange={setDeleting} />
    </>
  );
}
