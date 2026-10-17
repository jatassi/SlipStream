import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { Copy01Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { useRef, useState } from "react";

import { Button } from "./button";
import { dataFont } from "./data";

const copiedFor = 2000;

// The Clipboard API exists only in a secure context, and a Node is often
// opened over plain HTTP on the LAN, where a selected field still copies.
async function copy(input: HTMLInputElement) {
  if (isSecureContext) {
    await navigator.clipboard.writeText(input.value);
    return;
  }
  input.select();
  // oxlint-disable-next-line typescript/no-deprecated -- the one way to copy outside a secure context
  document.execCommand("copy");
}

function useCopied(): [copied: boolean, mark: () => void] {
  const [copied, setCopied] = useState(false);
  return [
    copied,
    () => {
      setCopied(true);
      setTimeout(() => {
        setCopied(false);
      }, copiedFor);
    },
  ];
}

export function ReadOnlyField({
  label,
  value,
  description,
}: {
  label: string;
  value: string;
  description: string;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [copied, markCopied] = useCopied();
  return (
    <Field.Root className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <div className="flex items-center gap-2">
        <Input
          ref={input}
          readOnly
          value={value}
          onFocus={(event) => {
            event.currentTarget.select();
          }}
          className={`h-(--input-height) min-w-0 flex-1 rounded-item bg-well px-3 ${dataFont} text-body text-ink-1 ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch`}
        />
        <Button
          aria-label={copied ? `Copied ${label}` : `Copy ${label}`}
          icon={copied ? Tick02Icon : Copy01Icon}
          onClick={() => {
            if (input.current) {
              void copy(input.current).then(markCopied);
            }
          }}
        >
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <Field.Description className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {description}
      </Field.Description>
    </Field.Root>
  );
}
