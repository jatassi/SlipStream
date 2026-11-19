import { Popover as BasePopover } from "@base-ui/react/popover";
import type { ReactNode } from "react";

import { labelPopup } from "./floating";
import { hoverOpening } from "./popover";
import { usePortalDensity } from "./portal-density";

/** One sentence that ends in a period, as a definition reads on hover or tap. */
export type Definition = `${string}.`;

/** A word a person reads, with what it means when it's a technical term (WRITING.md rule 25). */
export type Term = {
  readonly word: string;
  readonly definition?: Definition;
};

const definedWord = "underline decoration-ink-4 decoration-dotted underline-offset-3";

const trigger =
  "relative inline-flex rounded-chip before:absolute before:-inset-x-1 before:-inset-y-2 focus-visible:outline-2 focus-visible:outline-offset-2";

const definitionPopup = `max-w-[min(280px,var(--available-width))] text-pretty ${labelPopup}`;

/**
 * Draws a look around a term's word. A term with a definition underlines the
 * word with a dotted line and makes the whole look one button, which opens the
 * definition on hover, tap and Enter in the tooltip's glass label.
 */
export function Defined({
  term,
  children: look,
}: {
  term: Term;
  children: (word: ReactNode) => ReactNode;
}) {
  return term.definition === undefined ? (
    look(term.word)
  ) : (
    <Definable word={term.word} definition={term.definition}>
      {look(<span className={definedWord}>{term.word}</span>)}
    </Definable>
  );
}

function Definable({
  word,
  definition,
  children,
}: {
  word: string;
  definition: Definition;
  children: ReactNode;
}) {
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BasePopover.Root>
        <BasePopover.Trigger
          render={
            // oxlint-disable-next-line jsx-a11y/role-supports-aria-props -- ARIA 1.3's aria-description keeps the definition out of the row's text
            <button type="button" aria-description={definition} className={trigger}>
              {children}
            </button>
          }
          {...hoverOpening}
        />
        <BasePopover.Portal data-density={density}>
          <BasePopover.Positioner sideOffset={6} collisionPadding={12}>
            <BasePopover.Popup aria-label={word} initialFocus={false} className={definitionPopup}>
              {definition}
            </BasePopover.Popup>
          </BasePopover.Positioner>
        </BasePopover.Portal>
      </BasePopover.Root>
    </>
  );
}
