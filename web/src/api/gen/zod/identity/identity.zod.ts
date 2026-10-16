import * as zod from "zod";

import {
  EmailChangeInputBody,
  IdentityApiKeyMade,
  IdentityClaimBody,
  IdentityClaimed,
  IdentityEmailChangeStarted,
  IdentityIssuedResetLink,
  IdentityIssuedSignupLink,
  IdentityLinkView,
  IdentityPendingLinks,
  IdentityPinChange,
  IdentityReconfirmation,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
  IdentitySignedIn,
  IdentitySignupLinkURL,
  JoinInputBody,
  NewPINInputBody,
  NewSignupLinkInputBody,
  ResetLinkInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Change your email: a confirm link goes to the new address, and the email changes when it is followed
 */
export const StartEmailChangeBody = EmailChangeInputBody;

export const StartEmailChangeResponse = IdentityEmailChangeStarted;

/**
 * @summary Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN
 */
export const ChangePINBody = IdentityPinChange;

export const ChangePINResponse = zod.void();

/**
 * @summary Issue a Reset link for an Account, ending its sessions; the link is shown only in this answer
 */
export const CreateResetLinkParams = zod.object({
  id: zod.uuid(),
});

export const CreateResetLinkBody = ResetLinkInputBody;

export const CreateResetLinkResponse = IdentityIssuedResetLink;

/**
 * @summary Get when the API key was made
 */
export const GetAPIKeyResponse = IdentityApiKeyMade;

/**
 * @summary Replace the API key, shown this once; the old one stops working
 */
export const RegenerateAPIKeyResponse = IdentityRegeneratedAPIKey;

/**
 * @summary Get what a confirm link's page shows: the new email, or why it no longer works
 */
export const getEmailLinkByTokenPathTokenMax = 128;

export const GetEmailLinkByTokenParams = zod.object({
  token: zod.string().max(getEmailLinkByTokenPathTokenMax),
});

export const GetEmailLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Change an Account's email to the one a confirm link proved
 */
export const confirmEmailPathTokenMax = 128;

export const ConfirmEmailParams = zod.object({
  token: zod.string().max(confirmEmailPathTokenMax),
});

export const ConfirmEmailResponse = zod.void();

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
 * @summary Get what a Reset link's page shows: who it is from, or why it no longer works
 */
export const getResetLinkByTokenPathTokenMax = 128;

export const GetResetLinkByTokenParams = zod.object({
  token: zod.string().max(getResetLinkByTokenPathTokenMax),
});

export const GetResetLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Set a new PIN with a Reset link, ending every session, and sign in
 */
export const redeemResetLinkPathTokenMax = 128;

export const RedeemResetLinkParams = zod.object({
  token: zod.string().max(redeemResetLinkPathTokenMax),
});

export const RedeemResetLinkBody = NewPINInputBody;

export const RedeemResetLinkResponse = IdentitySignedIn;

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

export const CreateSessionResponse = IdentitySignedIn;

/**
 * @summary Confirm it's you with your PIN, for actions that change who can get in
 */
export const ReconfirmSessionBody = IdentityReconfirmation;

export const ReconfirmSessionResponse = zod.void();

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const ClaimControllerBody = IdentityClaimBody;

export const ClaimControllerResponse = IdentityClaimed;

/**
 * @summary List the Signup links nobody has used yet
 */
export const ListSignupLinksResponse = IdentityPendingLinks;

/**
 * @summary Issue a Signup link for an email, email it, and answer it to copy
 */
export const CreateSignupLinkBody = NewSignupLinkInputBody;

export const CreateSignupLinkResponse = IdentityIssuedSignupLink;

/**
 * @summary Get what a Signup link's page shows: who it is from, or why it no longer works
 */
export const getSignupLinkByTokenPathTokenMax = 128;

export const GetSignupLinkByTokenParams = zod.object({
  token: zod.string().max(getSignupLinkByTokenPathTokenMax),
});

export const GetSignupLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Make the Member's Account with a Signup link and sign in
 */
export const redeemSignupLinkPathTokenMax = 128;

export const RedeemSignupLinkParams = zod.object({
  token: zod.string().max(redeemSignupLinkPathTokenMax),
});

export const RedeemSignupLinkBody = JoinInputBody;

export const RedeemSignupLinkResponse = IdentitySignedIn;

/**
 * @summary Cancel a pending Signup link
 */
export const DeleteSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const DeleteSignupLinkResponse = zod.void();

/**
 * @summary Get a pending Signup link with its URL, to copy again
 */
export const GetSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const GetSignupLinkResponse = IdentitySignupLinkURL;

/**
 * @summary Send a pending Signup link again as a new link; the old one stops working
 */
export const ResendSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const ResendSignupLinkResponse = IdentityIssuedSignupLink;
