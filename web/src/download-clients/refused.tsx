import { errorText } from "../ui/form-context";
import { Refusal } from "../ui/setting-parts";

export function Refused({ errors }: { errors: readonly unknown[] }) {
  const error = errorText(errors);
  return error === undefined ? null : <Refusal error={error} />;
}
