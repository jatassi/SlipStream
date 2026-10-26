import { ContentTypeSwitches } from "../../works/content-type-switches";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

export function MediaStep({ nav }: StepBodyProps) {
  return (
    <>
      <ContentTypeSwitches />
      <StepFooter nav={nav} />
    </>
  );
}
