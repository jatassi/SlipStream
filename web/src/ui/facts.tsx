import { type ReactNode, createContext, use } from "react";

import { Button } from "./button";
import { signalLook } from "./signal-row";

/** Where the list sits, which sets its surface, rhythm and label ink. */
export type FactsLook = "sheet" | "well" | "inline" | "failure";

/**
 * The one action that changes a fact, such as Rename beside a Node's name.
 * name is what a screen reader says when the label alone, such as Change,
 * doesn't say which fact it changes.
 */
export type FactAction = {
  readonly label: string;
  readonly name?: string;
  readonly onClick: () => void;
};

/** A fact as data, for a component that takes its facts as an array. */
export type FactRow = { readonly label: string; readonly value: ReactNode };

type Look = {
  readonly list: string;
  readonly row: string;
  readonly label: string;
  readonly value: string;
};

const touchColumns = "in-data-[density=touch]:grid-cols-[minmax(0,1fr)_auto]";

const fitted = "grid-cols-[fit-content(200px)_minmax(0,1fr)_auto]";

const looks: { readonly [L in FactsLook]: Look } = {
  sheet: {
    list: "grid-cols-[minmax(120px,200px)_minmax(0,1fr)_auto] rounded-card bg-glass text-(length:--body-size) leading-(--body-line-height) ring-1 ring-hairline ring-inset",
    row: "relative min-h-(--row-height) items-center px-4 py-2 not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:min-h-(--row-height-two) in-data-[density=touch]:not-first:before:left-4",
    label: "text-ink-3",
    value: "text-ink-1",
  },
  well: {
    list: `${fitted} gap-y-1 rounded-item bg-well px-3 py-2 in-data-[density=touch]:gap-y-2`,
    row: "items-baseline",
    label: "text-ink-3",
    value: "text-ink-1",
  },
  inline: {
    list: `${fitted} gap-y-1.5 in-data-[density=touch]:gap-y-2`,
    row: "items-baseline",
    label: "text-ink-3",
    value: "text-ink-1",
  },
  failure: {
    list: "grid-cols-[max-content_minmax(0,78ch)_auto] gap-y-2",
    row: "items-baseline",
    label: "text-label-caps text-fail-ink uppercase",
    value: "text-fail-text",
  },
};

const LookContext = createContext<Look | undefined>(undefined);

type FactsProps = { readonly children: ReactNode } & (
  | { readonly look: "sheet" | "inline" | "failure" }
  | {
      readonly look: "well";
      /** The well sits on an amber or red row, so its labels take the row's dim ink and its values the row's text. */
      readonly signal?: keyof typeof signalLook;
    }
);

function lookOf(props: FactsProps): Look {
  const look = looks[props.look];
  if (props.look !== "well" || props.signal === undefined) {
    return look;
  }
  return { ...look, label: signalLook[props.signal].dim, value: "" };
}

/** Label and value rows whose labels line up down the list. In Touch each label sits over its value. */
export function Facts(props: FactsProps) {
  const look = lookOf(props);
  return (
    <dl className={`grid gap-x-4 ${look.list} ${touchColumns}`}>
      <LookContext value={look}>{props.children}</LookContext>
    </dl>
  );
}

const valueColumns = "col-[2/-1] in-data-[density=touch]:col-span-full";

export function Fact({
  label,
  action,
  children,
}: {
  /** Usually words. A file path is a label in Data. */
  readonly label: ReactNode;
  readonly action?: FactAction;
  /** The value as it reads: words, Data for a machine value, a Mark, a Defined term, a link or a control. */
  readonly children: ReactNode;
}) {
  const look = use(LookContext);
  if (look === undefined) {
    throw new Error("A Fact sits inside Facts");
  }
  return (
    <div
      className={`col-span-full grid grid-cols-subgrid in-data-[density=touch]:gap-y-0.5 ${look.row}`}
    >
      <dt className={`wrap-anywhere in-data-[density=touch]:col-span-full ${look.label}`}>
        {label}
      </dt>
      {action === undefined ? (
        <dd className={`min-w-0 wrap-anywhere ${valueColumns} ${look.value}`}>{children}</dd>
      ) : (
        <dd className={`grid grid-cols-subgrid items-center ${valueColumns} ${look.value}`}>
          <span className="min-w-0 wrap-anywhere">{children}</span>
          <Button size="small" look="ghost" aria-label={action.name} onClick={action.onClick}>
            {action.label}
          </Button>
        </dd>
      )}
    </div>
  );
}
