import { nounOf } from "../format";
import { withFieldGroup } from "../ui/form";
import { Segmented } from "../ui/segmented";
import { type Parts, partsSchema } from "./draft";

const units = [
  { value: "days", noun: { one: "day", many: "days" } },
  { value: "hours", noun: { one: "h", many: "h" } },
] as const;

const unitOptions = (amount: string) =>
  units.map(({ value, noun }) => ({ value, label: nounOf(Number(amount), noun) }));

const noParts: Parts = { ratio: "", amount: "", unit: "days" };

const check =
  (schema: (typeof partsSchema.shape)["ratio" | "amount"]) =>
  ({ value }: { value: string }) =>
    schema.safeParse(value).error?.issues[0]?.message;

/** A ratio and a seeding time, each shown only when what is edited counts it. */
export const PartsFields = withFieldGroup({
  defaultValues: noParts,
  props: { ratio: true, time: true, optional: false },
  render: function PartsRender({ group, ratio, time, optional }) {
    return (
      <div className="flex flex-wrap items-start gap-x-6 gap-y-3">
        {ratio && (
          <div className="w-32">
            <group.AppField name="ratio" validators={{ onSubmit: check(partsSchema.shape.ratio) }}>
              {(field) => <field.TextField label="Ratio" inputMode="decimal" autoComplete="off" />}
            </group.AppField>
          </div>
        )}
        {time && (
          <div className="flex flex-wrap items-start gap-2">
            <div className="w-32">
              <group.AppField
                name="amount"
                validators={{ onSubmit: check(partsSchema.shape.amount) }}
              >
                {(field) => (
                  <field.TextField label="Seeding time" inputMode="decimal" autoComplete="off" />
                )}
              </group.AppField>
            </div>
            <div className="pt-6.5">
              <group.Subscribe selector={(state) => state.values.amount}>
                {(amount) => (
                  <group.Field name="unit">
                    {(field) => (
                      <Segmented
                        label="Seeding time unit"
                        options={unitOptions(amount)}
                        value={field.state.value}
                        onValueChange={field.handleChange}
                      />
                    )}
                  </group.Field>
                )}
              </group.Subscribe>
            </div>
          </div>
        )}
        {optional && (
          <p className="basis-full text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            A goal counts the ratio, the seeding time or both. Leave one blank to count only the
            other.
          </p>
        )}
      </div>
    );
  },
});
