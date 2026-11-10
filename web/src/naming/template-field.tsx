import { useId } from "react";

import type {
  NamingFieldPreview,
  NamingToken,
  SettingTemplateField,
  WorksContentType,
} from "../api/gen/model";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Refusal } from "../ui/setting-parts";
import { inputLook, labelLook } from "../ui/setting-kind";
import type { DraftValue } from "./card";
import { TokenInput } from "./token-input";

function Recommended({ field, onUse }: { field: SettingTemplateField; onUse: () => void }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
      <p className="min-w-0 text-meta break-all text-ink-3 in-data-[density=touch]:text-body-touch">
        Recommended &quot;{field.default}&quot;.
      </p>
      <Button look="ghost" size="small" onClick={onUse}>
        Use recommended
      </Button>
    </div>
  );
}

/** The full paths the draft names for the type's samples, then what the save check refuses. */
function FieldPreview({ label, preview }: { label: string; preview: NamingFieldPreview }) {
  return (
    <>
      {preview.samples.length > 0 && (
        <ul aria-label={`${label} samples`} className="flex flex-col gap-1.5">
          {preview.samples.map((sample, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- two samples can name one path under one label
            <li key={index} className="flex flex-col">
              <span className="text-meta text-ink-3">{sample.label}</span>
              <Data className="break-all text-ink-1">{sample.path}</Data>
            </li>
          ))}
        </ul>
      )}
      {preview.problems.map((sentence) => (
        <Refusal key={sentence} error={sentence} />
      ))}
    </>
  );
}

/** The text box, with suggestions and Tokens when the template names tokens. */
function TemplateInput({
  type,
  field,
  value,
  tokens,
  invalid,
  onChange,
}: {
  type: WorksContentType;
  field: SettingTemplateField;
  value: string;
  tokens: readonly NamingToken[] | undefined;
  invalid: boolean;
  onChange: (value: string) => void;
}) {
  const id = useId();
  return (
    <>
      <label htmlFor={id} className={labelLook}>
        {field.label}
      </label>
      {tokens === undefined ? (
        <input
          id={id}
          aria-invalid={invalid || undefined}
          autoComplete="off"
          spellCheck={false}
          value={value}
          onChange={(event) => {
            onChange(event.target.value);
          }}
          className={`w-full ${inputLook}`}
        />
      ) : (
        <TokenInput {...{ id, type, value, tokens, invalid, onChange }} label={field.label} />
      )}
    </>
  );
}

/**
 * A template with Use recommended while it differs from the default, and
 * the draft's sample paths and problems under it.
 */
export function TemplateField({
  type,
  field,
  value,
  tokens,
  preview,
  onChange,
}: {
  type: WorksContentType;
  field: SettingTemplateField;
  value: string;
  /** Left out for a text that holds no tokens. */
  tokens: readonly NamingToken[] | undefined;
  preview: NamingFieldPreview | undefined;
  onChange: (value: DraftValue) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <TemplateInput
        type={type}
        field={field}
        value={value}
        tokens={tokens}
        invalid={(preview?.problems.length ?? 0) > 0}
        onChange={onChange}
      />
      {!field.plain && value !== field.default && (
        <Recommended
          field={field}
          onUse={() => {
            onChange(null);
          }}
        />
      )}
      {preview !== undefined && <FieldPreview label={field.label} preview={preview} />}
    </div>
  );
}
