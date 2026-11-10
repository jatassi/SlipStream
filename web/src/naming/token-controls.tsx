import { useId } from "react";

import type { NamingToken } from "../api/gen/model";
import { Select } from "../ui/select";
import { inputLook, labelLook } from "../ui/setting-kind";
import { Switch } from "../ui/switch";
import { type TokenChoice, cases, separators, takesWords, wraps } from "./token-text";

/** The labels of a modifier's text box and of its switch, when it has one. */
const modifierWords = {
  pad: { text: "Digits", invert: undefined, digits: true },
  cut: { text: "Longest, in bytes", invert: "Keep the end", digits: true },
  languages: { text: "Languages, such as EN+DE", invert: "Leave these out", digits: false },
  formats: {
    text: "Custom Formats, such as IMAX,HDR10+",
    invert: "Leave these out",
    digits: false,
  },
  format: { text: "Custom Format", invert: undefined, digits: false },
} as const;

type SetChoice = (part: Partial<TokenChoice>) => void;

function ModifierControls({
  token,
  choice,
  set,
}: {
  token: NamingToken;
  choice: TokenChoice;
  set: SetChoice;
}) {
  const id = useId();
  if (token.modifier === null) {
    return null;
  }
  const words = modifierWords[token.modifier];
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <label htmlFor={id} className={labelLook}>
          {words.text}
        </label>
        <input
          id={id}
          autoComplete="off"
          spellCheck={false}
          inputMode={words.digits ? "numeric" : "text"}
          value={choice.modifier}
          onChange={(event) => {
            const text = event.target.value;
            set({ modifier: words.digits ? text.replaceAll(/\D/gu, "") : text });
          }}
          className={`w-full ${inputLook}`}
        />
      </div>
      {words.invert !== undefined && (
        <div className="flex items-end">
          <Switch
            label={words.invert}
            checked={choice.invert}
            onCheckedChange={(invert) => {
              set({ invert });
            }}
          />
        </div>
      )}
    </>
  );
}

/** Case and the separator, shown as the token's example would print under each. */
function WordControls({
  token,
  choice,
  set,
}: {
  token: NamingToken;
  choice: TokenChoice;
  set: SetChoice;
}) {
  return (
    <>
      <Select
        label="Case"
        value={choice.case}
        options={cases}
        onValueChange={(next) => {
          set({ case: next });
        }}
      />
      {token.example.includes(" ") && (
        <Select
          label="Between words"
          value={choice.separator}
          options={separators.map((value) => ({
            value,
            label: token.example.replaceAll(" ", value),
          }))}
          onValueChange={(separator) => {
            set({ separator });
          }}
        />
      )}
    </>
  );
}

/** The controls a token offers: its tags, the text around it, its words and its modifier. */
export function TokenControls({
  token,
  choice,
  onChange,
}: {
  token: NamingToken;
  choice: TokenChoice;
  onChange: (choice: TokenChoice) => void;
}) {
  const set: SetChoice = (part) => {
    onChange({ ...choice, ...part });
  };
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      {token.tags.length > 0 && (
        <Select
          label="Tag"
          value={choice.tag}
          options={[
            { value: "", label: "None" },
            ...token.tags.map((tag) => ({ value: tag, label: `{${tag}-{${token.name}}}` })),
          ]}
          onValueChange={(tag) => {
            set({ tag });
          }}
        />
      )}
      {choice.tag === "" && (
        <Select
          label="Text around it"
          value={choice.wrap}
          options={wraps}
          onValueChange={(wrap) => {
            set({ wrap });
          }}
        />
      )}
      {takesWords(token) && <WordControls token={token} choice={choice} set={set} />}
      <ModifierControls token={token} choice={choice} set={set} />
    </div>
  );
}
