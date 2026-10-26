import { useNavigate, useRouteContext } from "@tanstack/react-router";
import { z } from "zod";

import { isProblem } from "../../api/errors";
import { claimController } from "../../api/gen/query/identity/identity";
import { ClaimControllerBody } from "../../api/gen/zod/identity/identity.zod";
import { markClaimed } from "../../session/session";
import { forgetSetupCode, setupCodeHeaders } from "../../session/setup-code";
import { submitErrors, useAppForm } from "../../ui/form";
import type { FormProps } from "./form-props";
import { busySentence } from "./join-sentences";

const startForm = ClaimControllerBody.extend({ pinAgain: z.string() }).refine(
  (value) => value.pin === value.pinAgain,
  { message: "The PINs don't match", path: ["pinAgain"] },
);

export type StartRefusal = "already_claimed" | "join_pending";

export function useStartForm(
  { code, restarted }: FormProps,
  refuse: (refusal: StartRefusal) => void,
) {
  const { queryClient } = useRouteContext({ from: "/setup" });
  const navigate = useNavigate();
  return useAppForm({
    defaultValues: { email: "", displayName: "", pin: "", pinAgain: "" },
    validators: { onSubmit: startForm },
    onSubmit: async ({ value, formApi }) => {
      const { pinAgain: _pinAgain, ...body } = startForm.parse(value);
      try {
        const signedIn = await claimController(body, { headers: setupCodeHeaders(code) });
        forgetSetupCode();
        markClaimed(queryClient, signedIn);
        await navigate({ to: "/" });
      } catch (error) {
        if (isProblem(error, "setup_code_wrong")) {
          restarted();
        } else if (isProblem(error, "already_claimed") || isProblem(error, "join_pending")) {
          refuse(error.code);
        } else {
          formApi.setErrorMap({
            onSubmit: isProblem(error, "setup_busy")
              ? { form: busySentence, fields: {} }
              : submitErrors(error),
          });
        }
      }
    },
  });
}
