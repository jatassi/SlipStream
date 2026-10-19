import "../ui/rows.css";

import {
  ArrowRight01Icon,
  CloudServerIcon,
  EyeIcon,
  Folder01Icon,
  FolderLibraryIcon,
  HardDriveIcon,
  LockIcon,
  UsbIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import type { ReactNode } from "react";

import type { RootsPlace, RootsPlaceKind } from "../api/gen/model";
import { bytes } from "../format";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Icon } from "../ui/icon";
import { VirtualListbox } from "../ui/virtual-listbox";
import { type Row, rowKey } from "./rows";

export const row = "flex items-center gap-3 px-4 text-left";

export const ruled =
  "shadow-[inset_0_1px_0_var(--color-hairline)] in-[[aria-posinset='1']]:shadow-none";

const placeIcons: { readonly [K in RootsPlaceKind]: IconSvgElement } = {
  local: HardDriveIcon,
  removable: UsbIcon,
  network: CloudServerIcon,
};

const placeChips: { readonly [K in RootsPlaceKind]: string | null } = {
  local: null,
  removable: "Removable",
  network: "Network",
};

function Line({
  icon,
  title,
  meta,
  trailing,
}: {
  icon: IconSvgElement;
  title: ReactNode;
  meta?: string;
  trailing?: ReactNode;
}) {
  return (
    <>
      <Icon icon={icon} className="size-4.5 shrink-0 text-ink-2" />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-body text-ink-1 in-data-[density=touch]:text-card-title">
          {title}
        </span>
        {meta !== undefined && <span className="truncate text-meta text-ink-3">{meta}</span>}
      </span>
      {trailing}
    </>
  );
}

const opens = <Icon icon={ArrowRight01Icon} className="size-4 shrink-0 text-ink-3" />;

function PlaceLine({ place }: { place: RootsPlace }) {
  const free = `${bytes(place.free)} free`;
  if (place.rootId !== undefined) {
    return (
      <Line
        icon={FolderLibraryIcon}
        title={<Data>{place.path}</Data>}
        meta={place.kind === undefined ? "Library root" : `Library root · ${free}`}
        trailing={opens}
      />
    );
  }
  const chip = place.kind === undefined ? null : placeChips[place.kind];
  return (
    <Line
      icon={place.kind === undefined ? HardDriveIcon : placeIcons[place.kind]}
      title={place.name}
      meta={`${free} of ${bytes(place.total)}`}
      trailing={
        <>
          {chip !== null && <Chip>{chip}</Chip>}
          {opens}
        </>
      }
    />
  );
}

function RowLine({ row: shown }: { row: Row }) {
  switch (shown.kind) {
    case "place":
      return <PlaceLine place={shown.place} />;
    case "folder":
      return shown.folder.readable ? (
        <Line icon={Folder01Icon} title={shown.folder.name} trailing={opens} />
      ) : (
        <Line
          icon={LockIcon}
          title={shown.folder.name}
          trailing={
            <>
              <span className="shrink-0 text-meta text-ink-3">Not allowed</span>
              {opens}
            </>
          }
        />
      );
    case "suggestion":
      return <Line icon={Folder01Icon} title={<Data>{shown.path}</Data>} trailing={opens} />;
    case "hidden":
      return (
        <Line
          icon={EyeIcon}
          title={<span className="text-ink-3">{shown.count} hidden</span>}
          trailing={
            <span aria-hidden="true" className="shrink-0 text-meta text-ink-3">
              Show
            </span>
          }
        />
      );
    default:
      return shown satisfies never;
  }
}

export function FolderRows({
  id,
  rows,
  active,
  estimate,
  onChoose,
}: {
  id: string;
  rows: readonly Row[];
  active: number;
  estimate: number;
  onChoose: (index: number) => void;
}) {
  return (
    <VirtualListbox
      id={id}
      label="Folders"
      rows={rows}
      rowKey={rowKey}
      estimate={estimate}
      active={active}
      onChoose={onChoose}
      renderRow={(shown, isActive) => (
        <div
          className={`${row} ${ruled} cursor-default transition-colors duration-350 ease-standard hover:bg-(--row-wash) ${shown.kind === "place" ? "min-h-(--row-height-two) py-2" : "min-h-(--row-height)"} ${isActive ? "bg-(--row-wash)" : ""}`}
        >
          <RowLine row={shown} />
        </div>
      )}
    />
  );
}
