import * as zod from "zod";

import {
  IdentityClaimBody,
  IdentityClaimed,
  IdentitySession,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get the sign-in limits and lifetimes, and the address links use
 */
export const GetIdentitySettingsResponse = IdentitySettings;

/**
 * @summary Change the sign-in limits and lifetimes, or the address links use
 */
export const UpdateIdentitySettingsBody = IdentitySettingsPatch;

export const UpdateIdentitySettingsResponse = IdentitySettings;

/**
 * @summary Sign out
 */
export const DeleteSessionResponse = zod.void();

/**
 * @summary Get the signed-in Account
 */
export const GetSessionResponse = IdentitySession;

/**
 * @summary Sign in with email and PIN
 */
export const CreateSessionBody = IdentitySignInBody;

export const CreateSessionResponse = IdentitySession;

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const ClaimControllerBody = IdentityClaimBody;

export const ClaimControllerResponse = IdentityClaimed;
