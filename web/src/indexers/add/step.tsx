import { CaptchaField, CodeField } from "../fields/captcha";
import { TestReport } from "./feed-fields";
import { type Source, formOf } from "./source";
import { type SignInStep, type Next, nextOf, type useAdd } from "./use-add";

type Add = ReturnType<typeof useAdd>;

/** The captcha shows while its step is current: shown and still for this sign-in, or expired. */
const captchaShown = (
  step: SignInStep,
  next: Next,
): step is Extract<SignInStep, { kind: "shown" | "expired" }> =>
  step.kind === "expired" || (step.kind === "shown" && next === "save");

/** The step after the fields: a Torznab feed's Test report, the captcha, or the two-factor code. */
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
        const definition = formOf(source);
        if (definition === undefined) {
          return null;
        }
        if (step.kind === "code") {
          return <CodeField form={form} fields={{ code: "code" }} name={definition.entry.name} />;
        }
        if (!captchaShown(step, next)) {
          return null;
        }
        return (
          <CaptchaField
            form={form}
            fields={{ captcha: "captcha" }}
            name={definition.entry.name}
            home={home}
            step={step}
            onReload={reload}
          />
        );
      }}
    </form.Subscribe>
  );
}
