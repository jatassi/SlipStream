import { Cancel01Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { useSuspenseInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";

import type { HistoryGroup } from "../api/gen/model";
import { useNow } from "../ui/clock";
import { Icon } from "../ui/icon";
import { IconButton } from "../ui/icon-button";
import { historyPages, kindsOptions } from "./history-query";
import { type HistoryFilter, paramsOf } from "./routed";

const chipLook =
  "inline-flex h-6 shrink-0 items-center gap-1 rounded-chip px-2 text-chip whitespace-nowrap transition-colors duration-(--duration-control) hover:text-ink-1 aria-pressed:bg-selected aria-pressed:text-ink-1 aria-pressed:shadow-[0_1px_3px_rgba(0,0,0,.28)] not-aria-pressed:text-ink-2";

/** A group is a filled chip. A kind is an outlined one, so it reads as part of the group it follows. */
const looks = {
  group: "not-aria-pressed:bg-fill",
  kind: "not-aria-pressed:ring-1 not-aria-pressed:ring-hairline-strong not-aria-pressed:ring-inset",
} as const;

type Change = (change: Partial<HistoryFilter>) => void;

function PressChip({
  look,
  pressed,
  onPress,
  children,
}: {
  look: keyof typeof looks;
  pressed: boolean;
  onPress: () => void;
  children: string;
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      className={`${chipLook} ${looks[look]}`}
      onClick={onPress}
    >
      {pressed && <Icon icon={Tick02Icon} className="size-3.5 shrink-0 text-glow" />}
      {children}
    </button>
  );
}

/** Each group as a chip; a chosen group's kinds follow it, each narrowing that group. */
export function GroupChips({ filter, onFilter }: { filter: HistoryFilter; onFilter: Change }) {
  const { groups } = useSuspenseQuery(kindsOptions()).data;
  const chosen = filter.groups ?? [];
  const kinds = filter.kinds ?? [];
  const toggleGroup = ({ group, kinds: inGroup }: HistoryGroup) => {
    if (!chosen.includes(group)) {
      onFilter({ groups: [...chosen, group] });
      return;
    }
    const dropped = new Set(inGroup.map(({ kind }) => kind));
    onFilter({
      groups: chosen.filter((each) => each !== group),
      kinds: kinds.filter((kind) => !dropped.has(kind)),
    });
  };
  return (
    <fieldset aria-label="Groups" className="flex min-w-0 flex-wrap gap-1.5">
      {groups.map((group) => (
        <span key={group.group} className="inline-flex flex-wrap gap-1">
          <PressChip
            look="group"
            pressed={chosen.includes(group.group)}
            onPress={() => {
              toggleGroup(group);
            }}
          >
            {group.label}
          </PressChip>
          {chosen.includes(group.group) &&
            group.kinds.map(({ kind, label }) => (
              <PressChip
                key={kind}
                look="kind"
                pressed={kinds.includes(kind)}
                onPress={() => {
                  onFilter({
                    kinds: kinds.includes(kind)
                      ? kinds.filter((each) => each !== kind)
                      : [...kinds, kind],
                  });
                }}
              >
                {label}
              </PressChip>
            ))}
        </span>
      ))}
    </fieldset>
  );
}

/** The Work History is narrowed to, by the label its entries carry while it exists. */
export function WorkChip({ filter, onFilter }: { filter: HistoryFilter; onFilter: Change }) {
  const { pages } = useSuspenseInfiniteQuery(historyPages(paramsOf(filter, useNow()))).data;
  const label =
    pages.flatMap((page) => page.entries).find((entry) => entry.work?.id === filter.work)?.work
      ?.label ?? "A deleted Work";
  return (
    <span className="inline-flex h-6 items-center gap-1 rounded-chip bg-selected pl-2 text-chip text-ink-1">
      {label}
      <IconButton
        icon={Cancel01Icon}
        label="Show every Work"
        className="size-6"
        onClick={() => {
          onFilter({ work: undefined });
        }}
      />
    </span>
  );
}
