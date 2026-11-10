import type {
  NamingFieldTokens,
  NamingPreview,
  SettingChoiceField,
  WorksContentType,
} from "../api/gen/model";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { Switch } from "../ui/switch";
import { type Draft, type NamingCard, customColon, edit, switchOf, textOf } from "./card";
import { TemplateField } from "./template-field";

function ChoiceField({
  field,
  draft,
  onChange,
}: {
  field: SettingChoiceField;
  draft: Draft;
  onChange: (draft: Draft) => void;
}) {
  return (
    <Select
      label={field.label}
      value={textOf(field, draft)}
      options={field.options}
      onValueChange={(value) => {
        onChange(edit(draft, field, value));
      }}
    />
  );
}

function RenameSwitch({
  type,
  card,
  draft,
  onChange,
}: {
  type: WorksContentType;
  card: NamingCard;
  draft: Draft;
  onChange: (draft: Draft) => void;
}) {
  const renaming = switchOf(card.rename, draft);
  return (
    <div className="flex flex-col gap-1">
      <Switch
        label={card.rename.label}
        checked={renaming}
        onCheckedChange={(value) => {
          onChange(edit(draft, card.rename, value));
        }}
      />
      {!renaming && (
        <p className="text-meta text-warn in-data-[density=touch]:text-body-touch">
          With renaming off, Jellyfin and Emby may show the files of a {type.work.one.toLowerCase()}{" "}
          as separate items. Imports keep the name of the downloaded file.
        </p>
      )}
    </div>
  );
}

export type CardFieldsProps = {
  type: WorksContentType;
  card: NamingCard;
  tokens: readonly NamingFieldTokens[];
  draft: Draft;
  preview: NamingPreview | undefined;
  busy: boolean;
  onChange: (draft: Draft) => void;
  onSave: () => void;
  onDiscard: () => void;
};

/** The card's Details: the rename switch, each template, the colon mode and the multi-episode style. */
export function CardFields({
  type,
  card,
  tokens,
  draft,
  preview,
  busy,
  onChange,
  onSave,
  onDiscard,
}: CardFieldsProps) {
  const previewOf = (key: string) => preview?.fields.find((field) => field.key === key);
  const template = (field: NamingCard["colonCustom"]) => (
    <TemplateField
      key={field.key}
      type={type}
      field={field}
      value={textOf(field, draft)}
      tokens={tokens.find((each) => each.key === field.key)?.tokens}
      preview={previewOf(field.key)}
      onChange={(value) => {
        onChange(edit(draft, field, value));
      }}
    />
  );
  return (
    <div className="mt-2 flex flex-col gap-5">
      <RenameSwitch type={type} card={card} draft={draft} onChange={onChange} />
      {card.templates.map((field) => template(field))}
      <ChoiceField field={card.colon} draft={draft} onChange={onChange} />
      {textOf(card.colon, draft) === customColon && template(card.colonCustom)}
      {card.multiEpisode !== undefined && (
        <ChoiceField field={card.multiEpisode} draft={draft} onChange={onChange} />
      )}
      {draft.size > 0 && (
        <div className="flex gap-2">
          <Button look="primary" busy={busy} onClick={onSave}>
            Save
          </Button>
          <Button look="ghost" disabled={busy} onClick={onDiscard}>
            Discard
          </Button>
        </div>
      )}
    </div>
  );
}
