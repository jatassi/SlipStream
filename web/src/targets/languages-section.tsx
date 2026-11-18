import { Add01Icon, Cancel01Icon, Tick02Icon } from "@hugeicons/core-free-icons";

import type { TargetsWantedLanguage as Want } from "../api/gen/model";
import {
  audioWords,
  duplicateWords,
  hearingWords,
  kindWords,
  languageName,
  subtitleLanguages,
  wantMeanings,
  wantPhrase,
  wantSentence,
} from "../subtitles/settings-words";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import type { SectionProps } from "./editor";
import { type ProfileDraft, duplicateOf } from "./profile-draft";

const tokenLook =
  "-mx-0.5 inline-flex items-center rounded-item px-1 py-0.5 font-medium text-ink-1 underline decoration-ink-3 decoration-dotted underline-offset-4 hover:bg-fill data-popup-open:bg-fill";

function Token<V extends string | boolean>({
  label,
  words,
  value,
  options,
  onChange,
}: {
  label: string;
  words: string;
  value: V;
  options: readonly { readonly value: V; readonly label: string }[];
  onChange: (value: V) => void;
}) {
  const items: MenuEntry[] = options.map((option) => ({
    label: option.label,
    icon: option.value === value ? Tick02Icon : undefined,
    onSelect: () => {
      onChange(option.value);
    },
  }));
  return (
    <Menu
      label={label}
      items={items}
      trigger={
        <button type="button" aria-label={`${label}: ${words}`} className={tokenLook}>
          {words}
        </button>
      }
    />
  );
}

const languageOptions = subtitleLanguages.map(({ code, name }) => ({ value: code, label: name }));

const hearingOptions = (["avoid", "allow", "prefer"] as const).map((value) => ({
  value,
  label: hearingWords[value],
}));

const kindOptions = [false, true].map((value) => ({ value, label: kindWords(value) }));

/** One Wanted Language as its sentence, each part a menu. */
function WantSentence({ want, onChange }: { want: Want; onChange: (next: Want) => void }) {
  const name = languageName(want.language);
  return (
    <p aria-label={wantSentence(want)} className="leading-8 text-ink-2">
      <Token
        label="When"
        words={audioWords(want)}
        value={want.audio}
        options={(["always", "no_audio"] as const).map((value) => ({
          value,
          label: audioWords({ audio: value, language: want.language }),
        }))}
        onChange={(audio) => {
          onChange({ ...want, audio });
        }}
      />
      {want.audio === "always" ? " get " : ", get "}
      <Token
        label="Language"
        words={name}
        value={want.language}
        options={languageOptions}
        onChange={(language) => {
          onChange({ ...want, language });
        }}
      />
      {" subtitles, "}
      <Token
        label="Kind"
        words={kindWords(want.forced)}
        value={want.forced}
        options={kindOptions}
        onChange={(forced) => {
          onChange({ ...want, forced });
        }}
      />
      {", hearing impaired "}
      <Token
        label="Hearing impaired"
        words={hearingWords[want.hearingImpaired]}
        value={want.hearingImpaired}
        options={hearingOptions}
        onChange={(hearingImpaired) => {
          onChange({ ...want, hearingImpaired });
        }}
      />
      .
    </p>
  );
}

const fresh = (language: string): Want => ({
  language,
  forced: false,
  hearingImpaired: "allow",
  audio: "always",
});

function WantRow({
  want,
  twice,
  onChange,
  onRemove,
}: {
  want: Want;
  twice: boolean;
  onChange: (next: Want) => void;
  onRemove: () => void;
}) {
  return (
    <li className="flex items-start gap-2 rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset">
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <WantSentence want={want} onChange={onChange} />
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {wantMeanings(want)}
        </p>
        {twice && (
          <p
            role="alert"
            className="text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
          >
            {duplicateWords(want)}
          </p>
        )}
      </div>
      <IconButton icon={Cancel01Icon} label={`Remove ${wantPhrase(want)}`} onClick={onRemove} />
    </li>
  );
}

function AddLanguage({ onAdd }: { onAdd: (want: Want) => void }) {
  return (
    <Menu
      label="Add a language"
      items={languageOptions.map((option) => ({
        label: option.label,
        onSelect: () => {
          onAdd(fresh(option.value));
        },
      }))}
      trigger={
        <Button size="small" icon={Add01Icon}>
          Add language
        </Button>
      }
    />
  );
}

/**
 * The Profile's Wanted Languages, each a sentence of menus, with what its
 * kind and hearing-impaired choice mean and an inline refusal for a
 * language and kind listed twice.
 */
export function LanguagesSection({ draft, onChange }: SectionProps<ProfileDraft>) {
  const duplicate = duplicateOf(draft.languages);
  const set = (languages: readonly Want[]) => {
    onChange({ ...draft, languages });
  };
  return (
    <div className="flex flex-col items-start gap-4">
      <p className="text-ink-2">
        The subtitles SlipStream gets for this Profile&apos;s files. An embedded text track or a
        subtitle file already beside a file counts.
      </p>
      {draft.languages.length > 0 && (
        <ul aria-label="Subtitle languages" className="flex flex-col gap-3 self-stretch">
          {draft.languages.map((want, index) => (
            <WantRow
              // oxlint-disable-next-line react/no-array-index-key -- two entries may name one language and kind until the refusal is fixed
              key={index}
              want={want}
              twice={duplicate === index}
              onChange={(next) => {
                set(draft.languages.map((each, at) => (at === index ? next : each)));
              }}
              onRemove={() => {
                set(draft.languages.filter((_, at) => at !== index));
              }}
            />
          ))}
        </ul>
      )}
      <AddLanguage
        onAdd={(want) => {
          set([...draft.languages, want]);
        }}
      />
    </div>
  );
}
