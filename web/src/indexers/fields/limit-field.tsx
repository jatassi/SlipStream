import { withFieldGroup } from "../../ui/form";
import { limitSchema } from "../edit-values";
import { Segmented } from "../../ui/segmented";

const windows = [
  { value: "hour", label: "per hour" },
  { value: "day", label: "per day" },
] as const;

export type LimitValues = { count: string; per: "hour" | "day" };

export const noLimit: LimitValues = { count: "0", per: "hour" };

/** A limit's count, where 0 is no limit, and its sliding window. */
export const LimitField = withFieldGroup({
  defaultValues: noLimit,
  props: { label: "Searches" },
  render: function Limit({ group, label }) {
    return (
      <div className="flex flex-wrap items-end gap-3">
        <div className="w-40">
          <group.AppField
            name="count"
            validators={{
              onSubmit: ({ value }) =>
                limitSchema.shape.count.safeParse(value).error?.issues[0]?.message,
            }}
          >
            {(field) => (
              <field.TextField label={label} inputMode="numeric" description="0 is no limit." />
            )}
          </group.AppField>
        </div>
        <div className="pb-6">
          <group.Field name="per">
            {(field) => (
              <Segmented
                label={`${label} window`}
                options={windows}
                value={field.state.value}
                onValueChange={field.handleChange}
              />
            )}
          </group.Field>
        </div>
      </div>
    );
  },
});
