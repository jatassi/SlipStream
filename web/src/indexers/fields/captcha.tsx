import { Refresh01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import type { IndexersCaptchaImage } from "../../api/gen/model";
import { clock } from "../../format";
import { Button } from "../../ui/button";
import { withFieldGroup } from "../../ui/form";
import type { CaptchaStep } from "../add/use-add";

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

const panel =
  "flex flex-col items-start gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset";

type Shown = Exclude<CaptchaStep, { kind: "none" }>;

type Reloads = { onReload?: (() => Promise<void>) | undefined };

function Expired({ words, onReload }: Reloads & { words: string }) {
  return (
    <div className={panel}>
      <p role="alert" className="text-ink-1">
        {words}
      </p>
      {onReload !== undefined && <Reload look="default" onReload={onReload} />}
    </div>
  );
}

function Image({
  name,
  home,
  captcha,
  onReload,
  children,
}: Reloads & {
  name: string;
  home: string;
  captcha: IndexersCaptchaImage;
  children: ReactNode;
}) {
  return (
    <div className={panel}>
      <p className="text-ink-1">
        Signing in to {name} needs the code in this image, which {home} fetched. It expires at{" "}
        {clock(captcha.expiresAt)}.
      </p>
      <img
        src={captcha.image}
        alt={`The captcha ${name} shows`}
        className="max-h-24 max-w-full rounded-item bg-solid object-contain p-1"
      />
      <div className="flex w-full flex-wrap items-end gap-2">
        <div className="min-w-0 flex-[1_1_16rem]">{children}</div>
        {onReload !== undefined && (
          <div className="pb-0.5">
            <Reload look="ghost" onReload={onReload} />
          </div>
        )}
      </div>
    </div>
  );
}

const props: Reloads & { name: string; home: string; step: Shown } = {
  name: "",
  home: "",
  step: { kind: "expired", signIn: "", words: "" },
};

/**
 * The captcha step of a sign-in that shows one: the image the Node fetched
 * and the code typed from it, or, once the attempt has expired, a way to
 * load a new one. Without onReload, the form's next submit loads it.
 */
export const CaptchaField = withFieldGroup({
  defaultValues: { captcha: "" },
  props,
  render: function Captcha({ group, name, home, step, onReload }) {
    if (step.kind === "expired") {
      return <Expired words={step.words} onReload={onReload} />;
    }
    return (
      <Image name={name} home={home} captcha={step.captcha} onReload={onReload}>
        <group.AppField
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
        </group.AppField>
      </Image>
    );
  },
});
