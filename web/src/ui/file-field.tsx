import { FileUploadIcon } from "@hugeicons/core-free-icons";
import { useId, useRef, useState } from "react";

import { Button } from "./button";
import { Data } from "./data";

type Props = {
  label: string;
  /** The file types the picker offers, as the input's accept attribute. */
  accept: string;
  description?: string;
  /** Takes the picked file, so the caller reads it. */
  onFile: (file: File) => Promise<void> | void;
};

function Description({ id, text }: { id: string; text: string | undefined }) {
  return text === undefined ? null : (
    <p id={id} className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
      {text}
    </p>
  );
}

/** A file picked from the device: its label, Choose a file, and the chosen file's name. */
export function FileField({ label, accept, description, onFile }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const [chosen, setChosen] = useState<string>();
  const descriptionId = useId();
  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="mb-1.5 text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </legend>
      <div className="flex min-w-0 items-center gap-3">
        <Button
          icon={FileUploadIcon}
          aria-describedby={description === undefined ? undefined : descriptionId}
          onClick={() => {
            input.current?.click();
          }}
        >
          Choose a file
        </Button>
        {chosen !== undefined && <Data className="truncate text-ink-2">{chosen}</Data>}
      </div>
      <input
        ref={input}
        type="file"
        accept={accept}
        aria-label={label}
        tabIndex={-1}
        className="sr-only"
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = "";
          if (file !== undefined) {
            setChosen(file.name);
            void onFile(file);
          }
        }}
      />
      <Description id={descriptionId} text={description} />
    </fieldset>
  );
}
