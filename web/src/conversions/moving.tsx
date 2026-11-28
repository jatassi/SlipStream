import { ArrowUp02Icon, DragDropVerticalIcon } from "@hugeicons/core-free-icons";
import { type DragEvent, useState } from "react";

import type { ConversionsConversionRow } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { IconButton } from "../ui/icon-button";
import { moveAfter } from "./actions";

const dragType = "application/x-slipstream-conversion";

/** Dropping a dragged Conversion on a row puts it right after that row, in that row's lane. */
export function useDrop(row: ConversionsConversionRow) {
  const [over, setOver] = useState(false);
  return {
    over,
    handlers: {
      onDragOver: (event: DragEvent) => {
        if (event.dataTransfer.types.includes(dragType)) {
          event.preventDefault();
          setOver(true);
        }
      },
      onDragLeave: () => {
        setOver(false);
      },
      onDrop: (event: DragEvent) => {
        setOver(false);
        const dragged = event.dataTransfer.getData(dragType);
        if (dragged !== "" && dragged !== row.id) {
          void moveAfter(dragged, row.id);
        }
      },
    },
  };
}

/** The pointer's way to move a row; Move up is the keyboard's. */
export function Handle({ row }: { row: ConversionsConversionRow }) {
  return (
    <span
      draggable
      aria-hidden="true"
      title="Drag to move"
      className="flex cursor-grab text-ink-4"
      onDragStart={(event) => {
        event.dataTransfer.setData(dragType, row.id);
        event.dataTransfer.effectAllowed = "move";
      }}
    >
      <Icon icon={DragDropVerticalIcon} />
    </span>
  );
}

export function MoveUpButton({ subject, onMoveUp }: { subject: string; onMoveUp: () => void }) {
  return <IconButton icon={ArrowUp02Icon} label={`Move up ${subject}`} onClick={onMoveUp} />;
}
