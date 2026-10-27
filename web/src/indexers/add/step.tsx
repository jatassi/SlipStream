import { CaptchaField } from "../fields/captcha";
import { TestReport } from "./feed-fields";
import type { Source } from "./source";
import { type CaptchaStep, type Next, nextOf, type useAdd } from "./use-add";

type Add = ReturnType<typeof useAdd>;

/** The captcha shows while its step is current: shown and still for this sign-in, or expired. */
const captchaShown = (
  step: CaptchaStep,
  next: Next,
): step is Exclude<CaptchaStep, { kind: "none" }> =>
  step.kind === "expired" || (step.kind === "shown" && next === "save");

/** The step after the fields: a Torznab feed's Test report, or the captcha. */
export function Step({ add, source, home }: { add: Add; source: Source; home: string }) {
  const { form, step, tried, reload } = add;
  return (
    <form.Subscribe selector={(state) => nextOf(source, state.values, step, tried)}>
      {(next) => {
        if (source.kind === "torznab") {
          return next === "test" || tried === undefined ? null : (
            <TestReport form={form} tried={tried} />
          );
        }
        if (source.kind !== "catalog" || !captchaShown(step, next)) {
          return null;
        }
        return (
          <CaptchaField
            form={form}
            name={source.form.entry.name}
            home={home}
            step={step}
            onReload={reload}
          />
        );
      }}
    </form.Subscribe>
  );
}
