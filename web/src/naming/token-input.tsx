import {
  type KeyboardEvent,
  type RefObject,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import type { NamingToken, WorksContentType } from "../api/gen/model";
import { moveIndex } from "../ui/move-index";
import { inputLook } from "../ui/setting-kind";
import { VirtualListbox, optionId } from "../ui/virtual-listbox";
import { type Suggestions, pick, suggest } from "./suggest";
import { TokenBuilder } from "./token-builder";

const keys: Readonly<Record<string, "next" | "prev">> = { ArrowDown: "next", ArrowUp: "prev" };

/** Where the next insert goes: the caret or the selection the text box last held. */
type Selection = { readonly start: number; readonly end: number };

/** The suggestions as "{" is typed: open from a keystroke until a pick, Escape or blur. */
function useSuggesting(
  input: RefObject<HTMLInputElement | null>,
  tokens: readonly NamingToken[],
  value: string,
  replace: (text: string, caret: number) => void,
) {
  const [typedAt, setTypedAt] = useState<number>();
  const [active, setActive] = useState(0);
  const suggestions = typedAt === undefined ? undefined : suggest(tokens, value, typedAt);
  const choose = (index: number) => {
    const token = suggestions?.tokens[index];
    if (suggestions !== undefined && token !== undefined) {
      const next = pick(value, suggestions, token);
      setTypedAt(undefined);
      replace(next.text, next.caret);
    }
  };
  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (suggestions === undefined) {
      return;
    }
    const move = keys[event.key];
    if (move !== undefined) {
      setActive((from) => moveIndex(from, move, suggestions.tokens.length, 1));
    } else if (event.key === "Enter" || event.key === "Tab") {
      choose(active);
    } else if (event.key === "Escape") {
      setTypedAt(undefined);
      event.stopPropagation();
    } else {
      return;
    }
    event.preventDefault();
  };
  const typed = () => {
    setTypedAt(input.current?.selectionStart ?? undefined);
    setActive(0);
  };
  const close = () => {
    setTypedAt(undefined);
  };
  return { suggestions, active, choose, onKeyDown, typed, close };
}

function SuggestionList({
  id,
  suggestions,
  active,
  onChoose,
}: {
  id: string;
  suggestions: Suggestions;
  active: number;
  onChoose: (index: number) => void;
}) {
  return (
    <div className="absolute inset-x-0 top-full z-10 mt-1.5 flex max-h-64 flex-col overflow-hidden rounded-card bg-popover p-1 text-ink-1 shadow-float backdrop-blur-[30px] backdrop-saturate-[1.7]">
      <VirtualListbox
        id={id}
        label="Tokens"
        rows={suggestions.tokens}
        rowKey={(token) => token.name}
        estimate={36}
        active={active}
        onChoose={onChoose}
        renderRow={(token, isActive) => (
          <span
            className={`flex h-9 cursor-default items-center justify-between gap-3 rounded-item px-3 select-none ${isActive ? "bg-fill-hover" : ""}`}
          >
            <span className="shrink-0">{token.name}</span>
            <span className="truncate text-meta text-ink-3">{token.example}</span>
          </span>
        )}
      />
    </div>
  );
}

/** The selection an insert replaces, and the caret a replace leaves once the new value renders. */
function useCaret(
  input: RefObject<HTMLInputElement | null>,
  value: string,
  onChange: (value: string) => void,
) {
  const selection = useRef<Selection>({ start: value.length, end: value.length });
  const placed = useRef<number>(undefined);
  useLayoutEffect(() => {
    if (placed.current !== undefined) {
      input.current?.setSelectionRange(placed.current, placed.current);
      placed.current = undefined;
    }
  });
  const replace = (text: string, caret: number) => {
    placed.current = caret;
    selection.current = { start: caret, end: caret };
    onChange(text);
  };
  return {
    replace,
    insert: (text: string) => {
      const { start, end } = selection.current;
      replace(value.slice(0, start) + text + value.slice(end), start + text.length);
    },
    select: (next: Selection) => {
      selection.current = next;
    },
  };
}

type ComboProps = {
  input: RefObject<HTMLInputElement | null>;
  id: string;
  listId: string;
  open: boolean;
  active: number;
  invalid: boolean;
  value: string;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  onChange: (value: string) => void;
  onSelect: (selection: Selection) => void;
  onBlur: () => void;
};

function Combo({ input, id, listId, open, active, invalid, value, ...on }: ComboProps) {
  return (
    <input
      ref={input}
      id={id}
      role="combobox"
      aria-autocomplete="list"
      aria-expanded={open}
      aria-controls={open ? listId : undefined}
      aria-activedescendant={open ? optionId(listId, active) : undefined}
      aria-invalid={invalid || undefined}
      autoComplete="off"
      spellCheck={false}
      value={value}
      onChange={(event) => {
        const { value: next, selectionStart, selectionEnd } = event.target;
        on.onSelect({ start: selectionStart ?? next.length, end: selectionEnd ?? next.length });
        on.onChange(next);
      }}
      onSelect={(event) => {
        const { selectionStart, selectionEnd } = event.currentTarget;
        on.onSelect({ start: selectionStart ?? 0, end: selectionEnd ?? 0 });
      }}
      onKeyDown={on.onKeyDown}
      onBlur={on.onBlur}
      className={`w-full ${inputLook}`}
    />
  );
}

/**
 * A template's text box. Typing "{" lists the tokens that fit what follows
 * it, and Tokens opens the builder, which inserts at the caret.
 */
export function TokenInput({
  id,
  type,
  label,
  value,
  tokens,
  invalid,
  onChange,
}: {
  id: string;
  type: WorksContentType;
  label: string;
  value: string;
  tokens: readonly NamingToken[];
  invalid: boolean;
  onChange: (value: string) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const listId = useId();
  const caret = useCaret(input, value, onChange);
  const { suggestions, active, choose, onKeyDown, typed, close } = useSuggesting(
    input,
    tokens,
    value,
    caret.replace,
  );
  return (
    <div className="flex items-center gap-2">
      <div className="relative min-w-0 flex-1">
        <Combo
          {...{ input, id, listId, active, invalid, value, onKeyDown }}
          open={suggestions !== undefined}
          onChange={(next) => {
            typed();
            onChange(next);
          }}
          onSelect={caret.select}
          onBlur={close}
        />
        {suggestions !== undefined && (
          <SuggestionList id={listId} suggestions={suggestions} active={active} onChoose={choose} />
        )}
      </div>
      <TokenBuilder {...{ type, label, tokens }} onInsert={caret.insert} />
    </div>
  );
}
