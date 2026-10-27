import { Refresh01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { clock } from "../../format";
import { Button } from "../../ui/button";
import type { AddForm, CaptchaStep } from "../add/use-add";

function Reload({ look, onReload }: { look: "default" | "ghost"; onReload: () => Promise<void> }) {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      look={look}
      icon={Refresh01Icon}
      busy={busy}
      onClick={() => {
        setBusy(true);
        void onReload().finally(() => {
          setBusy(false);
        });
      }}
    >
      New captcha
    </Button>
  );
}

function CodeField({ form }: { form: AddForm }) {
  return (
    <form.AppField
      name="captcha"
      validators={{
        onSubmit: ({ value }) =>
          value.trim() === "" ? "Type the code the image shows." : undefined,
      }}
    >
      {(field) => (
        <field.TextField
          label="Code"
          autoComplete="off"
          autoCapitalize="off"
          spellCheck={false}
          // oxlint-disable-next-line jsx-a11y/no-autofocus -- the code is the only thing left to type once the captcha shows
          autoFocus
        />
      )}
    </form.AppField>
  );
}

const panel =
  "flex flex-col items-start gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset";

/**
 * The captcha step of a definition whose sign-in shows one: the image the
 * home Node fetched and the code typed from it, or, once the attempt has
 * expired, a way to load a new one.
 */
export function CaptchaField({
  form,
  name,
  home,
  step,
  onReload,
}: {
  form: AddForm;
  name: string;
  home: string;
  step: Exclude<CaptchaStep, { kind: "none" }>;
  onReload: () => Promise<void>;
}) {
  if (step.kind === "expired") {
    return (
      <div className={panel}>
        <p role="alert" className="text-ink-1">
          {step.words}
        </p>
        <Reload look="default" onReload={onReload} />
      </div>
    );
  }
  return (
    <div className={panel}>
      <p className="text-ink-1">
        Signing in to {name} needs the code in this image, which {home} fetched. It expires at{" "}
        {clock(step.captcha.expiresAt)}.
      </p>
      <img
        src={step.captcha.image}
        alt={`The captcha ${name} shows`}
        className="max-h-24 max-w-full rounded-item bg-solid object-contain p-1"
      />
      <div className="flex w-full flex-wrap items-end gap-2">
        <div className="min-w-0 flex-[1_1_16rem]">
          <CodeField form={form} />
        </div>
        <div className="pb-0.5">
          <Reload look="ghost" onReload={onReload} />
        </div>
      </div>
    </div>
  );
}
