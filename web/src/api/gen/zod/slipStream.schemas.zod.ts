import * as zod from "zod";

export const emailChangeInputBodyEmailMax = 254;

export const EmailChangeInputBody = zod.object({
  email: zod.email().max(emailChangeInputBodyEmailMax),
});

export type EmailChangeInputBody = zod.input<typeof EmailChangeInputBody>;
export type EmailChangeInputBodyOutput = zod.output<typeof EmailChangeInputBody>;

export const FieldError = zod.object({
  location: zod.string(),
  message: zod.string(),
});

export type FieldError = zod.input<typeof FieldError>;
export type FieldErrorOutput = zod.output<typeof FieldError>;

export const FilesMissingCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["missing"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesMissingCell = zod.input<typeof FilesMissingCell>;
export type FilesMissingCellOutput = zod.output<typeof FilesMissingCell>;

export const FilesDownloadingCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["downloading"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesDownloadingCell = zod.input<typeof FilesDownloadingCell>;
export type FilesDownloadingCellOutput = zod.output<typeof FilesDownloadingCell>;

export const FilesFailedCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["failed"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesFailedCell = zod.input<typeof FilesFailedCell>;
export type FilesFailedCellOutput = zod.output<typeof FilesFailedCell>;

export const filesProbeHeightMin = 0;

export const filesProbeWidthMin = 0;

export const FilesProbe = zod.object({
  height: zod.int().min(filesProbeHeightMin),
  width: zod.int().min(filesProbeWidthMin),
});

export type FilesProbe = zod.input<typeof FilesProbe>;
export type FilesProbeOutput = zod.output<typeof FilesProbe>;

export const filesLibraryFileSizeMin = 0;

export const FilesLibraryFile = zod.object({
  id: zod.uuid(),
  path: zod.string().describe("Relative to its root, / separated"),
  probe: FilesProbe,
  size: zod.int().min(filesLibraryFileSizeMin),
});

export type FilesLibraryFile = zod.input<typeof FilesLibraryFile>;
export type FilesLibraryFileOutput = zod.output<typeof FilesLibraryFile>;

export const FilesMetCell = zod.object({
  edition: zod.enum(["standard"]),
  file: FilesLibraryFile.describe("The file filling it"),
  itemId: zod.uuid(),
  state: zod.enum(["met"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesMetCell = zod.input<typeof FilesMetCell>;
export type FilesMetCellOutput = zod.output<typeof FilesMetCell>;

export const FilesCell = zod.discriminatedUnion("state", [
  FilesMissingCell,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesMetCell,
]);

export type FilesCell = zod.input<typeof FilesCell>;
export type FilesCellOutput = zod.output<typeof FilesCell>;

export const FilesCellList = zod.object({
  cells: zod.array(FilesCell).describe("By Item in tree order, then Target in Profile order"),
});

export type FilesCellList = zod.input<typeof FilesCellList>;
export type FilesCellListOutput = zod.output<typeof FilesCellList>;

export const FilesChanged = zod.object({
  workId: zod.uuid(),
});

export type FilesChanged = zod.input<typeof FilesChanged>;
export type FilesChangedOutput = zod.output<typeof FilesChanged>;

export const GrabsChanged = zod.object({
  grabId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsChanged = zod.input<typeof GrabsChanged>;
export type GrabsChangedOutput = zod.output<typeof GrabsChanged>;

export const GrabsEntered = zod.object({
  at: zod.iso.datetime({ offset: true }),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
});

export type GrabsEntered = zod.input<typeof GrabsEntered>;
export type GrabsEnteredOutput = zod.output<typeof GrabsEntered>;

export const GrabsFailure = zod.object({
  code: zod.enum([
    "indexer_unavailable",
    "torrent_unreachable",
    "torrent_invalid",
    "infohash_mismatch",
    "torrent_v2_only",
    "no_download_client",
    "download_failed",
    "no_video_file",
    "probe_failed",
    "fills_nothing",
    "transfer_failed",
    "import_failed",
  ]),
  message: zod.string(),
  stage: zod.enum(["fetching", "downloading", "probing", "transferring", "importing"]),
});

export type GrabsFailure = zod.input<typeof GrabsFailure>;
export type GrabsFailureOutput = zod.output<typeof GrabsFailure>;

export const grabsProgressDoneMin = 0;

export const grabsProgressTotalMin = 0;

export const GrabsProgress = zod.object({
  done: zod
    .int()
    .min(grabsProgressDoneMin)
    .describe("Bytes downloaded, or received and verified while transferring"),
  total: zod.int().min(grabsProgressTotalMin).describe("Bytes wanted"),
});

export type GrabsProgress = zod.input<typeof GrabsProgress>;
export type GrabsProgressOutput = zod.output<typeof GrabsProgress>;

export const grabsGrabReleaseSizeMin = 0;

export const GrabsGrab = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  edition: zod.enum(["standard"]),
  failure: GrabsFailure.exactOptional().describe("Why it failed, absent unless it did"),
  history: zod.array(GrabsEntered).describe("Each stage it entered, oldest first"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  infoHash: zod
    .string()
    .nullable()
    .describe("The v1 info hash in hex, null until the .torrent is fetched"),
  itemId: zod.uuid(),
  nodeId: zod.uuid().describe("The Node that downloads it: the Indexer's home Node"),
  progress: GrabsProgress.exactOptional().describe(
    "The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring",
  ),
  releaseSize: zod.int().min(grabsGrabReleaseSizeMin),
  releaseTitle: zod.string(),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
  targetIds: zod.array(zod.string()).describe("The Targets it was grabbed for, in Profile order"),
  workId: zod.uuid(),
});

export type GrabsGrab = zod.input<typeof GrabsGrab>;
export type GrabsGrabOutput = zod.output<typeof GrabsGrab>;

export const GrabsGrabList = zod.object({
  grabs: zod.array(GrabsGrab).describe("Newest first"),
});

export type GrabsGrabList = zod.input<typeof GrabsGrabList>;
export type GrabsGrabListOutput = zod.output<typeof GrabsGrabList>;

export const grabsSearchCandidatesMin = 0;

export const GrabsSearch = zod.object({
  candidates: zod.int().min(grabsSearchCandidatesMin).describe("The releases the Indexer answered"),
  finishedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while the run searches"),
  grabId: zod.uuid().nullable().describe("The Grab the run made, null when none"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  itemId: zod.uuid(),
  outcome: zod
    .union([
      zod.literal("grabbed"),
      zod.literal("nothing_acceptable"),
      zod.literal("indexer_unavailable"),
      zod.literal("indexer_refused"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while the run searches"),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type GrabsSearch = zod.input<typeof GrabsSearch>;
export type GrabsSearchOutput = zod.output<typeof GrabsSearch>;

export const GrabsSearchChanged = zod.object({
  searchId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsSearchChanged = zod.input<typeof GrabsSearchChanged>;
export type GrabsSearchChangedOutput = zod.output<typeof GrabsSearchChanged>;

export const GrabsSearchList = zod.object({
  searches: zod.array(GrabsSearch).describe("Newest first"),
});

export type GrabsSearchList = zod.input<typeof GrabsSearchList>;
export type GrabsSearchListOutput = zod.output<typeof GrabsSearchList>;

export const GrabsSearchStarted = zod.object({
  searchIds: zod.array(zod.string()).describe("One run per Item with a wanted cell"),
});

export type GrabsSearchStarted = zod.input<typeof GrabsSearchStarted>;
export type GrabsSearchStartedOutput = zod.output<typeof GrabsSearchStarted>;

export const identityAccountDisplayNameMax = 100;

export const identityAccountEmailMax = 254;

export const IdentityAccount = zod.object({
  admin: zod.boolean(),
  displayName: zod.string().min(1).max(identityAccountDisplayNameMax),
  email: zod.email().max(identityAccountEmailMax),
  id: zod.uuid(),
});

export type IdentityAccount = zod.input<typeof IdentityAccount>;
export type IdentityAccountOutput = zod.output<typeof IdentityAccount>;

export const IdentityAccountsChanged = zod.object({});

export type IdentityAccountsChanged = zod.input<typeof IdentityAccountsChanged>;
export type IdentityAccountsChangedOutput = zod.output<typeof IdentityAccountsChanged>;

export const IdentityApiKeyMade = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityApiKeyMade = zod.input<typeof IdentityApiKeyMade>;
export type IdentityApiKeyMadeOutput = zod.output<typeof IdentityApiKeyMade>;

export const identityClaimBodyDisplayNameMax = 100;

export const identityClaimBodyEmailMax = 254;

export const identityClaimBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityClaimBody = zod.object({
  displayName: zod.string().min(1).max(identityClaimBodyDisplayNameMax),
  email: zod.email().max(identityClaimBodyEmailMax),
  pin: zod.string().regex(identityClaimBodyPinRegExp),
});

export type IdentityClaimBody = zod.input<typeof IdentityClaimBody>;
export type IdentityClaimBodyOutput = zod.output<typeof IdentityClaimBody>;

export const identityClaimedApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityClaimed = zod.object({
  account: IdentityAccount,
  apiKey: zod.string().regex(identityClaimedApiKeyRegExp),
});

export type IdentityClaimed = zod.input<typeof IdentityClaimed>;
export type IdentityClaimedOutput = zod.output<typeof IdentityClaimed>;

export const IdentityDeadLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe(
      "Who to ask: absent for unknown, which names nobody, and for a link the API key or the host command issued",
    ),
  expiresAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Every reason but unknown"),
  issuedAt: zod.iso.datetime({ offset: true }).exactOptional().describe("Every reason but unknown"),
  reason: zod.enum(["unknown", "withdrawn", "expired", "used"]),
  state: zod.enum(["dead"]),
});

export type IdentityDeadLink = zod.input<typeof IdentityDeadLink>;
export type IdentityDeadLinkOutput = zod.output<typeof IdentityDeadLink>;

export const MailOutcome = zod.object({
  detail: zod
    .string()
    .exactOptional()
    .describe("The mail server's reply, or why none answered, when the email failed"),
  outcome: zod.enum(["sent", "failed", "not-set-up"]),
});

export type MailOutcome = zod.input<typeof MailOutcome>;
export type MailOutcomeOutput = zod.output<typeof MailOutcome>;

export const IdentityEmailChangeStarted = zod.object({
  email: MailOutcome.describe("What became of the confirm link's email to the new address"),
});

export type IdentityEmailChangeStarted = zod.input<typeof IdentityEmailChangeStarted>;
export type IdentityEmailChangeStartedOutput = zod.output<typeof IdentityEmailChangeStarted>;

export const IdentityIssuedResetLink = zod.object({
  email: MailOutcome,
  url: zod.url().describe("The link itself, shown only in this answer"),
});

export type IdentityIssuedResetLink = zod.input<typeof IdentityIssuedResetLink>;
export type IdentityIssuedResetLinkOutput = zod.output<typeof IdentityIssuedResetLink>;

export const IdentityPendingLink = zod.object({
  email: zod.email(),
  expiresAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
  issuedAt: zod.iso.datetime({ offset: true }),
  issuedBy: zod
    .string()
    .exactOptional()
    .describe("The Admin who issued it; absent for the API key"),
});

export type IdentityPendingLink = zod.input<typeof IdentityPendingLink>;
export type IdentityPendingLinkOutput = zod.output<typeof IdentityPendingLink>;

export const IdentityIssuedSignupLink = zod.object({
  email: MailOutcome,
  link: IdentityPendingLink,
  url: zod.url().describe("The link itself, to copy"),
});

export type IdentityIssuedSignupLink = zod.input<typeof IdentityIssuedSignupLink>;
export type IdentityIssuedSignupLinkOutput = zod.output<typeof IdentityIssuedSignupLink>;

export const IdentityLiveLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe("Who issued the link; absent when the API key or the host command did"),
  email: zod
    .email()
    .describe("Where the link went: the Account's email, or the new one it confirms"),
  expiresAt: zod.iso.datetime({ offset: true }),
  serverName: zod.string(),
  state: zod.enum(["live"]),
});

export type IdentityLiveLink = zod.input<typeof IdentityLiveLink>;
export type IdentityLiveLinkOutput = zod.output<typeof IdentityLiveLink>;

export const IdentityLinkView = zod.discriminatedUnion("state", [
  IdentityLiveLink,
  IdentityDeadLink,
]);

export type IdentityLinkView = zod.input<typeof IdentityLinkView>;
export type IdentityLinkViewOutput = zod.output<typeof IdentityLinkView>;

export const IdentityLinksChanged = zod.object({});

export type IdentityLinksChanged = zod.input<typeof IdentityLinksChanged>;
export type IdentityLinksChangedOutput = zod.output<typeof IdentityLinksChanged>;

export const IdentityPendingLinks = zod.object({
  links: zod.array(IdentityPendingLink),
});

export type IdentityPendingLinks = zod.input<typeof IdentityPendingLinks>;
export type IdentityPendingLinksOutput = zod.output<typeof IdentityPendingLinks>;

export const identityPinChangeCurrentRegExp = new RegExp("^[0-9]{4}$");
export const identityPinChangeNewRegExp = new RegExp("^[0-9]{4}$");

export const IdentityPinChange = zod.object({
  current: zod
    .string()
    .regex(identityPinChangeCurrentRegExp)
    .exactOptional()
    .describe("The current PIN, left out only by a session that must set a new PIN"),
  new: zod.string().regex(identityPinChangeNewRegExp),
});

export type IdentityPinChange = zod.input<typeof IdentityPinChange>;
export type IdentityPinChangeOutput = zod.output<typeof IdentityPinChange>;

export const identityReconfirmationPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityReconfirmation = zod.object({
  pin: zod.string().regex(identityReconfirmationPinRegExp),
});

export type IdentityReconfirmation = zod.input<typeof IdentityReconfirmation>;
export type IdentityReconfirmationOutput = zod.output<typeof IdentityReconfirmation>;

export const identityRegeneratedAPIKeyApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityRegeneratedAPIKey = zod.object({
  apiKey: zod.string().regex(identityRegeneratedAPIKeyApiKeyRegExp),
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityRegeneratedAPIKey = zod.input<typeof IdentityRegeneratedAPIKey>;
export type IdentityRegeneratedAPIKeyOutput = zod.output<typeof IdentityRegeneratedAPIKey>;

export const IdentitySession = zod.object({
  account: IdentityAccount,
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySession = zod.input<typeof IdentitySession>;
export type IdentitySessionOutput = zod.output<typeof IdentitySession>;

export const IdentitySessionsChanged = zod.object({
  account: zod.uuid(),
});

export type IdentitySessionsChanged = zod.input<typeof IdentitySessionsChanged>;
export type IdentitySessionsChangedOutput = zod.output<typeof IdentitySessionsChanged>;

export const IdentitySettings = zod.object({
  address: zod.string().describe("The origin links are built on, empty until set"),
  addressFailures: zod.int().describe("Failed sign-in attempts one address may make in the window"),
  addressWindowSeconds: zod.int(),
  emailLinkLifetimeSeconds: zod.int(),
  pinOffAfter: zod.int().describe("Wrong PINs in a row before the PIN turns off"),
  pinWaitAfter: zod.int().describe("Wrong PINs before a wait"),
  pinWaitSeconds: zod.int().describe("The wait after too many wrong PINs"),
  reconfirmWindowSeconds: zod.int().describe("How long a confirm with the PIN or a passkey lasts"),
  resetLinkLifetimeSeconds: zod.int(),
  sessionLifetimeSeconds: zod.int().describe("How long a session lasts unused"),
  signupLinkLifetimeSeconds: zod.int(),
  suggestedAddress: zod
    .string()
    .describe(
      "The address this request reached SlipStream at, or this machine's name when that was an IP address",
    ),
});

export type IdentitySettings = zod.input<typeof IdentitySettings>;
export type IdentitySettingsOutput = zod.output<typeof IdentitySettings>;

export const identitySettingsPatchAddressMax = 2048;

export const IdentitySettingsPatch = zod.object({
  address: zod.string().max(identitySettingsPatchAddressMax).exactOptional(),
  addressFailures: zod.int().exactOptional(),
  addressWindowSeconds: zod.int().exactOptional(),
  emailLinkLifetimeSeconds: zod.int().exactOptional(),
  pinOffAfter: zod.int().exactOptional(),
  pinWaitAfter: zod.int().exactOptional(),
  pinWaitSeconds: zod.int().exactOptional(),
  reconfirmWindowSeconds: zod.int().exactOptional(),
  resetLinkLifetimeSeconds: zod.int().exactOptional(),
  sessionLifetimeSeconds: zod.int().exactOptional(),
  signupLinkLifetimeSeconds: zod.int().exactOptional(),
});

export type IdentitySettingsPatch = zod.input<typeof IdentitySettingsPatch>;
export type IdentitySettingsPatchOutput = zod.output<typeof IdentitySettingsPatch>;

export const identitySignInBodyEmailMax = 254;

export const identitySignInBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentitySignInBody = zod.object({
  email: zod.email().max(identitySignInBodyEmailMax),
  pin: zod.string().regex(identitySignInBodyPinRegExp),
});

export type IdentitySignInBody = zod.input<typeof IdentitySignInBody>;
export type IdentitySignInBodyOutput = zod.output<typeof IdentitySignInBody>;

export const IdentitySignedIn = zod.object({
  account: IdentityAccount,
  passkeyPrompt: zod
    .boolean()
    .describe("Show the prompt to add a passkey for this address, once in this session"),
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySignedIn = zod.input<typeof IdentitySignedIn>;
export type IdentitySignedInOutput = zod.output<typeof IdentitySignedIn>;

export const IdentitySignupLinkURL = zod.object({
  link: IdentityPendingLink,
  url: zod.url(),
});

export type IdentitySignupLinkURL = zod.input<typeof IdentitySignupLinkURL>;
export type IdentitySignupLinkURLOutput = zod.output<typeof IdentitySignupLinkURL>;

export const IndexersChanged = zod.object({
  indexerId: zod.uuid(),
});

export type IndexersChanged = zod.input<typeof IndexersChanged>;
export type IndexersChangedOutput = zod.output<typeof IndexersChanged>;

export const IndexersSearchMode = zod.object({
  mode: zod.string().describe("The Torznab t= value"),
  params: zod.array(zod.string()).describe("In the Indexer's order"),
});

export type IndexersSearchMode = zod.input<typeof IndexersSearchMode>;
export type IndexersSearchModeOutput = zod.output<typeof IndexersSearchMode>;

export const IndexersIndexer = zod.object({
  apiKeySet: zod
    .boolean()
    .describe("False when the key was dropped and an Admin must enter it again"),
  createdAt: zod.iso.datetime({ offset: true }),
  homeNodeId: zod.uuid().describe("The Node that reaches the Indexer"),
  id: zod.uuid(),
  name: zod.string(),
  protocol: zod.enum(["torznab"]),
  searchModes: zod.array(IndexersSearchMode).describe("By mode"),
  url: zod.string(),
});

export type IndexersIndexer = zod.input<typeof IndexersIndexer>;
export type IndexersIndexerOutput = zod.output<typeof IndexersIndexer>;

export const IndexersIndexerList = zod.object({
  indexers: zod.array(IndexersIndexer).describe("By name"),
});

export type IndexersIndexerList = zod.input<typeof IndexersIndexerList>;
export type IndexersIndexerListOutput = zod.output<typeof IndexersIndexerList>;

export const indexersNewAPIKeyApiKeyMax = 256;

export const IndexersNewAPIKey = zod.object({
  apiKey: zod.string().max(indexersNewAPIKeyApiKeyMax),
});

export type IndexersNewAPIKey = zod.input<typeof IndexersNewAPIKey>;
export type IndexersNewAPIKeyOutput = zod.output<typeof IndexersNewAPIKey>;

export const indexersNewIndexerApiKeyMax = 256;

export const indexersNewIndexerNameMax = 100;

export const indexersNewIndexerUrlMax = 2048;

export const indexersNewIndexerUrlRegExp = new RegExp("^https?://[^/?#]+(/[^?#]*)?$");

export const IndexersNewIndexer = zod.object({
  apiKey: zod.string().max(indexersNewIndexerApiKeyMax),
  name: zod.string().min(1).max(indexersNewIndexerNameMax),
  url: zod
    .string()
    .max(indexersNewIndexerUrlMax)
    .regex(indexersNewIndexerUrlRegExp)
    .describe("The Torznab feed's base; SlipStream adds /api"),
});

export type IndexersNewIndexer = zod.input<typeof IndexersNewIndexer>;
export type IndexersNewIndexerOutput = zod.output<typeof IndexersNewIndexer>;

export const joinInputBodyDisplayNameMax = 100;

export const joinInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const JoinInputBody = zod.object({
  displayName: zod.string().min(1).max(joinInputBodyDisplayNameMax),
  pin: zod.string().regex(joinInputBodyPinRegExp),
});

export type JoinInputBody = zod.input<typeof JoinInputBody>;
export type JoinInputBodyOutput = zod.output<typeof JoinInputBody>;

export const LiveAccessChanged = zod.object({});

export type LiveAccessChanged = zod.input<typeof LiveAccessChanged>;
export type LiveAccessChangedOutput = zod.output<typeof LiveAccessChanged>;

export const LiveResync = zod.object({});

export type LiveResync = zod.input<typeof LiveResync>;
export type LiveResyncOutput = zod.output<typeof LiveResync>;

export const LiveSessionEnded = zod.object({
  reason: zod.enum(["signed-out", "expired", "revoked"]),
});

export type LiveSessionEnded = zod.input<typeof LiveSessionEnded>;
export type LiveSessionEndedOutput = zod.output<typeof LiveSessionEnded>;

export const MailSettings = zod.object({
  fromAddress: zod.string(),
  fromName: zod.string(),
  host: zod.string().describe("Empty until email is set up"),
  passwordSet: zod.boolean().describe("Whether a password is saved; it is never shown"),
  port: zod.int().min(1),
  security: zod.enum(["none", "starttls", "tls"]),
  username: zod.string().describe("Empty when the server takes no sign-in"),
});

export type MailSettings = zod.input<typeof MailSettings>;
export type MailSettingsOutput = zod.output<typeof MailSettings>;

export const mailSettingsPatchFromAddressMax = 254;

export const mailSettingsPatchFromNameMax = 100;

export const mailSettingsPatchHostMax = 253;

export const mailSettingsPatchPasswordMax = 1024;

export const mailSettingsPatchUsernameMax = 254;

export const MailSettingsPatch = zod.object({
  fromAddress: zod.string().max(mailSettingsPatchFromAddressMax).exactOptional(),
  fromName: zod.string().max(mailSettingsPatchFromNameMax).exactOptional(),
  host: zod.string().max(mailSettingsPatchHostMax).exactOptional(),
  password: zod
    .string()
    .max(mailSettingsPatchPasswordMax)
    .exactOptional()
    .describe("Write-only; empty clears it"),
  port: zod.int().min(1).exactOptional(),
  security: zod.enum(["none", "starttls", "tls"]).exactOptional(),
  username: zod.string().max(mailSettingsPatchUsernameMax).exactOptional(),
});

export type MailSettingsPatch = zod.input<typeof MailSettingsPatch>;
export type MailSettingsPatchOutput = zod.output<typeof MailSettingsPatch>;

export const newPINInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const NewPINInputBody = zod.object({
  pin: zod.string().regex(newPINInputBodyPinRegExp),
});

export type NewPINInputBody = zod.input<typeof NewPINInputBody>;
export type NewPINInputBodyOutput = zod.output<typeof NewPINInputBody>;

export const newSignupLinkInputBodyEmailMax = 254;

export const NewSignupLinkInputBody = zod.object({
  email: zod.email().max(newSignupLinkInputBodyEmailMax),
});

export type NewSignupLinkInputBody = zod.input<typeof NewSignupLinkInputBody>;
export type NewSignupLinkInputBodyOutput = zod.output<typeof NewSignupLinkInputBody>;

export const NodesBuild = zod.object({
  arch: zod.string(),
  installKind: zod.string(),
  os: zod.string(),
  version: zod.string(),
});

export type NodesBuild = zod.input<typeof NodesBuild>;
export type NodesBuildOutput = zod.output<typeof NodesBuild>;

export const NodesChanged = zod.object({
  nodeId: zod.uuid(),
});

export type NodesChanged = zod.input<typeof NodesChanged>;
export type NodesChangedOutput = zod.output<typeof NodesChanged>;

export const NodesFacts = zod.object({
  appliedVersion: zod.string(),
  assigned: zod.array(zod.string()),
  restarting: zod.array(zod.string()),
  running: zod.array(zod.string()),
});

export type NodesFacts = zod.input<typeof NodesFacts>;
export type NodesFactsOutput = zod.output<typeof NodesFacts>;

export const NodesJoinCode = zod.object({
  code: zod.string(),
  expiresAt: zod.iso.datetime({ offset: true }),
});

export type NodesJoinCode = zod.input<typeof NodesJoinCode>;
export type NodesJoinCodeOutput = zod.output<typeof NodesJoinCode>;

export const NodesLink = zod.object({
  build: NodesBuild.exactOptional().describe("Absent while the link is down"),
  since: zod.iso.datetime({ offset: true }),
  state: zod.enum(["local", "up", "down", "mismatched"]),
});

export type NodesLink = zod.input<typeof NodesLink>;
export type NodesLinkOutput = zod.output<typeof NodesLink>;

export const nodesNodeIntentVersionRegExp = new RegExp("^[0-9a-f]{32}$");

export const NodesNode = zod.object({
  addresses: zod.array(zod.string()),
  facts: NodesFacts.exactOptional().describe("Absent while the Node has no session"),
  id: zod.uuid(),
  intentVersion: zod.string().regex(nodesNodeIntentVersionRegExp),
  key: zod.string().describe("The key's fingerprint"),
  link: NodesLink,
  name: zod.string(),
  pairedAt: zod.iso.datetime({ offset: true }),
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
  self: zod.boolean(),
});

export type NodesNode = zod.input<typeof NodesNode>;
export type NodesNodeOutput = zod.output<typeof NodesNode>;

export const NodesNodeList = zod.object({
  nodes: zod.array(NodesNode),
});

export type NodesNodeList = zod.input<typeof NodesNodeList>;
export type NodesNodeListOutput = zod.output<typeof NodesNodeList>;

export const NodesRoles = zod.object({
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
});

export type NodesRoles = zod.input<typeof NodesRoles>;
export type NodesRolesOutput = zod.output<typeof NodesRoles>;

export const nodesSettingsJoinCodeLifetimeSecondsMin = 600;
export const nodesSettingsJoinCodeLifetimeSecondsMax = 604800;

export const NodesSettings = zod.object({
  joinCodeLifetimeSeconds: zod
    .int()
    .min(nodesSettingsJoinCodeLifetimeSecondsMin)
    .max(nodesSettingsJoinCodeLifetimeSecondsMax),
});

export type NodesSettings = zod.input<typeof NodesSettings>;
export type NodesSettingsOutput = zod.output<typeof NodesSettings>;

export const nodesSettingsPatchJoinCodeLifetimeSecondsMin = 600;
export const nodesSettingsPatchJoinCodeLifetimeSecondsMax = 604800;

export const NodesSettingsPatch = zod.object({
  joinCodeLifetimeSeconds: zod
    .int()
    .min(nodesSettingsPatchJoinCodeLifetimeSecondsMin)
    .max(nodesSettingsPatchJoinCodeLifetimeSecondsMax)
    .exactOptional(),
});

export type NodesSettingsPatch = zod.input<typeof NodesSettingsPatch>;
export type NodesSettingsPatchOutput = zod.output<typeof NodesSettingsPatch>;

export const PairingController = zod.object({
  key: zod.string().describe("The Controller key's fingerprint"),
  link: zod.string(),
  name: zod.string(),
});

export type PairingController = zod.input<typeof PairingController>;
export type PairingControllerOutput = zod.output<typeof PairingController>;

export const pairingJoinBodyCodeMax = 1024;

export const PairingJoinBody = zod.object({
  code: zod.string().min(1).max(pairingJoinBodyCodeMax),
});

export type PairingJoinBody = zod.input<typeof PairingJoinBody>;
export type PairingJoinBodyOutput = zod.output<typeof PairingJoinBody>;

export const PairingWorker = zod.object({
  controller: PairingController,
  nodeId: zod.uuid(),
});

export type PairingWorker = zod.input<typeof PairingWorker>;
export type PairingWorkerOutput = zod.output<typeof PairingWorker>;

export const Problem = zod.object({
  code: zod.string(),
  detail: zod.string().exactOptional(),
  errors: zod.array(FieldError).exactOptional(),
  status: zod.int(),
  title: zod.string(),
});

export type Problem = zod.input<typeof Problem>;
export type ProblemOutput = zod.output<typeof Problem>;

export const ResetLinkInputBody = zod.object({
  removePasskeys: zod
    .boolean()
    .exactOptional()
    .describe("Also remove the Account's passkeys, for when someone else may have had access"),
});

export type ResetLinkInputBody = zod.input<typeof ResetLinkInputBody>;
export type ResetLinkInputBodyOutput = zod.output<typeof ResetLinkInputBody>;

export const RootsChanged = zod.object({});

export type RootsChanged = zod.input<typeof RootsChanged>;
export type RootsChangedOutput = zod.output<typeof RootsChanged>;

export const rootsNewRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const rootsNewRootPathMax = 4096;

export const RootsNewRoot = zod.object({
  contentType: zod.string().regex(rootsNewRootContentTypeRegExp),
  path: zod
    .string()
    .min(1)
    .max(rootsNewRootPathMax)
    .describe("An absolute path to an existing folder on the Controller"),
});

export type RootsNewRoot = zod.input<typeof RootsNewRoot>;
export type RootsNewRootOutput = zod.output<typeof RootsNewRoot>;

export const rootsRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const RootsRoot = zod.object({
  contentType: zod.string().regex(rootsRootContentTypeRegExp),
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The Node whose disk holds it"),
  path: zod.string(),
});

export type RootsRoot = zod.input<typeof RootsRoot>;
export type RootsRootOutput = zod.output<typeof RootsRoot>;

export const RootsRootList = zod.object({
  roots: zod.array(RootsRoot).describe("By Content Type"),
});

export type RootsRootList = zod.input<typeof RootsRootList>;
export type RootsRootListOutput = zod.output<typeof RootsRootList>;

export const shellCountsActivityMin = 0;

export const ShellCounts = zod.object({
  activity: zod.int().min(shellCountsActivityMin).describe("The Grabs neither imported nor failed"),
});

export type ShellCounts = zod.input<typeof ShellCounts>;
export type ShellCountsOutput = zod.output<typeof ShellCounts>;

export const SystemBuild = zod.object({
  commit: zod.string().exactOptional(),
  fingerprint: zod.string().exactOptional(),
  libtorrent: zod.string().exactOptional(),
  nativeBundle: zod.string().exactOptional(),
  stamped: zod.boolean(),
  target: zod.string().exactOptional(),
  version: zod.string().exactOptional(),
});

export type SystemBuild = zod.input<typeof SystemBuild>;
export type SystemBuildOutput = zod.output<typeof SystemBuild>;

export const SystemHealth = zod.object({
  status: zod.enum(["ok"]),
});

export type SystemHealth = zod.input<typeof SystemHealth>;
export type SystemHealthOutput = zod.output<typeof SystemHealth>;

export const SystemListeners = zod.object({
  control: zod.string(),
  http: zod.string(),
  link: zod.string(),
  torrent: zod.string(),
});

export type SystemListeners = zod.input<typeof SystemListeners>;
export type SystemListenersOutput = zod.output<typeof SystemListeners>;

export const SystemReport = zod.object({
  build: SystemBuild,
  dataDir: zod.string(),
  listeners: SystemListeners,
  pid: zod.int(),
  seed: zod.string(),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type SystemReport = zod.input<typeof SystemReport>;
export type SystemReportOutput = zod.output<typeof SystemReport>;

export const SystemStatus = zod.object({
  claim: zod.enum(["unclaimed", "controller", "worker"]),
});

export type SystemStatus = zod.input<typeof SystemStatus>;
export type SystemStatusOutput = zod.output<typeof SystemStatus>;

export const TargetsChanged = zod.object({});

export type TargetsChanged = zod.input<typeof TargetsChanged>;
export type TargetsChangedOutput = zod.output<typeof TargetsChanged>;

export const targetsNewProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const targetsNewProfileNameMax = 100;

export const TargetsNewProfile = zod.object({
  contentType: zod.string().regex(targetsNewProfileContentTypeRegExp),
  name: zod.string().min(1).max(targetsNewProfileNameMax),
  targetIds: zod.array(zod.uuid()).min(1).describe("Highest ranked first"),
});

export type TargetsNewProfile = zod.input<typeof TargetsNewProfile>;
export type TargetsNewProfileOutput = zod.output<typeof TargetsNewProfile>;

export const targetsNewTargetNameMax = 100;

export const TargetsNewTarget = zod.object({
  name: zod.string().min(1).max(targetsNewTargetNameMax),
  resolutions: zod
    .array(zod.enum(["2160p", "1080p", "720p", "576p", "480p"]))
    .min(1)
    .describe("Highest ranked first"),
});

export type TargetsNewTarget = zod.input<typeof TargetsNewTarget>;
export type TargetsNewTargetOutput = zod.output<typeof TargetsNewTarget>;

export const targetsProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const TargetsProfile = zod.object({
  contentType: zod.string().regex(targetsProfileContentTypeRegExp),
  default: zod.boolean().describe("Whether the Content Type's Works use it"),
  id: zod.uuid(),
  name: zod.string(),
  targetIds: zod.array(zod.uuid()).describe("Highest ranked first"),
});

export type TargetsProfile = zod.input<typeof TargetsProfile>;
export type TargetsProfileOutput = zod.output<typeof TargetsProfile>;

export const TargetsProfileList = zod.object({
  profiles: zod.array(TargetsProfile).describe("By Content Type, then name"),
});

export type TargetsProfileList = zod.input<typeof TargetsProfileList>;
export type TargetsProfileListOutput = zod.output<typeof TargetsProfileList>;

export const TargetsTarget = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
  name: zod.string(),
  resolutions: zod
    .array(zod.enum(["2160p", "1080p", "720p", "576p", "480p"]))
    .describe("Highest ranked first"),
});

export type TargetsTarget = zod.input<typeof TargetsTarget>;
export type TargetsTargetOutput = zod.output<typeof TargetsTarget>;

export const TargetsTargetList = zod.object({
  targets: zod.array(TargetsTarget).describe("By name"),
});

export type TargetsTargetList = zod.input<typeof TargetsTargetList>;
export type TargetsTargetListOutput = zod.output<typeof TargetsTargetList>;

export const WorksTextValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["text"]),
  text: zod.string(),
});

export type WorksTextValue = zod.input<typeof WorksTextValue>;
export type WorksTextValueOutput = zod.output<typeof WorksTextValue>;

export const WorksNumberValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["number"]),
  number: zod.number(),
});

export type WorksNumberValue = zod.input<typeof WorksNumberValue>;
export type WorksNumberValueOutput = zod.output<typeof WorksNumberValue>;

export const worksDateValueDateRegExp = new RegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$");

export const WorksDateValue = zod.object({
  date: zod.string().regex(worksDateValueDateRegExp).describe("YYYY, YYYY-MM or YYYY-MM-DD"),
  key: zod.string(),
  kind: zod.enum(["date"]),
});

export type WorksDateValue = zod.input<typeof WorksDateValue>;
export type WorksDateValueOutput = zod.output<typeof WorksDateValue>;

export const worksDurationValueMsMin = 0;

export const WorksDurationValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["duration"]),
  ms: zod.int().min(worksDurationValueMsMin),
});

export type WorksDurationValue = zod.input<typeof WorksDurationValue>;
export type WorksDurationValueOutput = zod.output<typeof WorksDurationValue>;

export const WorksFieldValue = zod.discriminatedUnion("kind", [
  WorksTextValue,
  WorksNumberValue,
  WorksDateValue,
  WorksDurationValue,
]);

export type WorksFieldValue = zod.input<typeof WorksFieldValue>;
export type WorksFieldValueOutput = zod.output<typeof WorksFieldValue>;

export const WorksItem = zod.object({
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  levelId: zod.uuid().nullable().describe("Null when the Content Type declares no levels"),
  monitored: zod.boolean(),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksItem = zod.input<typeof WorksItem>;
export type WorksItemOutput = zod.output<typeof WorksItem>;

export const WorksLevel = zod.object({
  depth: zod.int().min(1),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  parentId: zod.uuid().nullable().describe("Null under the Work"),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksLevel = zod.input<typeof WorksLevel>;
export type WorksLevelOutput = zod.output<typeof WorksLevel>;

export const worksTreeContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksTree = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksTreeContentTypeRegExp),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  items: zod
    .array(WorksItem)
    .describe("By their level's order, live before withdrawn, then position"),
  key: zod.string(),
  label: zod.string(),
  levels: zod
    .array(WorksLevel)
    .describe("By depth, then their parent's order, live before withdrawn, then position"),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksTree = zod.input<typeof WorksTree>;
export type WorksTreeOutput = zod.output<typeof WorksTree>;

export const WorksAdded = zod.object({
  created: zod.boolean().describe("False when the Work was already in the library"),
  work: WorksTree,
});

export type WorksAdded = zod.input<typeof WorksAdded>;
export type WorksAddedOutput = zod.output<typeof WorksAdded>;

export const WorksCandidate = zod.object({
  fields: zod.array(WorksFieldValue).describe("The Work's fields"),
  key: zod.string(),
  label: zod.string(),
  workId: zod.uuid().nullable().describe("The Work, once it is in the library"),
});

export type WorksCandidate = zod.input<typeof WorksCandidate>;
export type WorksCandidateOutput = zod.output<typeof WorksCandidate>;

export const WorksCandidateList = zod.object({
  candidates: zod.array(WorksCandidate).describe("Best first"),
});

export type WorksCandidateList = zod.input<typeof WorksCandidateList>;
export type WorksCandidateListOutput = zod.output<typeof WorksCandidateList>;

export const WorksFieldSpec = zod.object({
  depth: zod
    .int()
    .min(1)
    .exactOptional()
    .describe("The level's depth, 1 for the outermost; present only on tier level"),
  key: zod.string(),
  kind: zod.enum(["text", "number", "date", "duration"]),
  label: zod.string(),
  tier: zod.enum(["work", "level", "item"]),
});

export type WorksFieldSpec = zod.input<typeof WorksFieldSpec>;
export type WorksFieldSpecOutput = zod.output<typeof WorksFieldSpec>;

export const WorksNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type WorksNoun = zod.input<typeof WorksNoun>;
export type WorksNounOutput = zod.output<typeof WorksNoun>;

export const worksContentTypeIdRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentType = zod.object({
  enabled: zod.boolean(),
  fields: zod.array(WorksFieldSpec),
  icon: zod.enum(["film", "tv", "note", "open-book", "headphones", "panels", "image", "shapes"]),
  id: zod.string().regex(worksContentTypeIdRegExp),
  item: WorksNoun,
  levels: zod.array(WorksNoun).describe("Outermost first; empty when Items hang off the Work"),
  name: zod.string(),
  work: WorksNoun,
});

export type WorksContentType = zod.input<typeof WorksContentType>;
export type WorksContentTypeOutput = zod.output<typeof WorksContentType>;

export const WorksContentTypeList = zod.object({
  contentTypes: zod.array(WorksContentType),
});

export type WorksContentTypeList = zod.input<typeof WorksContentTypeList>;
export type WorksContentTypeListOutput = zod.output<typeof WorksContentTypeList>;

export const worksContentTypesChangedContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentTypesChanged = zod.object({
  contentType: zod.string().regex(worksContentTypesChangedContentTypeRegExp),
});

export type WorksContentTypesChanged = zod.input<typeof WorksContentTypesChanged>;
export type WorksContentTypesChangedOutput = zod.output<typeof WorksContentTypesChanged>;

export const WorksEnabled = zod.object({
  enabled: zod.boolean(),
});

export type WorksEnabled = zod.input<typeof WorksEnabled>;
export type WorksEnabledOutput = zod.output<typeof WorksEnabled>;

export const worksListedWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksListedWork = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksListedWorkContentTypeRegExp),
  fields: zod.array(WorksFieldValue).describe("The Work's own fields"),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksListedWork = zod.input<typeof WorksListedWork>;
export type WorksListedWorkOutput = zod.output<typeof WorksListedWork>;

export const worksNewWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksNewWorkKeyMax = 200;

export const WorksNewWork = zod.object({
  contentType: zod.string().regex(worksNewWorkContentTypeRegExp),
  key: zod
    .string()
    .min(1)
    .max(worksNewWorkKeyMax)
    .describe("The Work's key at its metadata source"),
  monitor: zod
    .boolean()
    .exactOptional()
    .describe("Monitor a new Work's Items; adding a Work already in the library changes nothing"),
});

export type WorksNewWork = zod.input<typeof WorksNewWork>;
export type WorksNewWorkOutput = zod.output<typeof WorksNewWork>;

export const WorksWorkChanged = zod.object({
  workId: zod.uuid(),
});

export type WorksWorkChanged = zod.input<typeof WorksWorkChanged>;
export type WorksWorkChangedOutput = zod.output<typeof WorksWorkChanged>;

export const WorksWorkList = zod.object({
  works: zod.array(WorksListedWork).describe("By sort label, then id"),
});

export type WorksWorkList = zod.input<typeof WorksWorkList>;
export type WorksWorkListOutput = zod.output<typeof WorksWorkList>;

export const LiveEventFilesChanged = zod.object({
  event: zod.enum(["files.changed"]),
  data: FilesChanged,
});

export type LiveEventFilesChanged = zod.input<typeof LiveEventFilesChanged>;
export type LiveEventFilesChangedOutput = zod.output<typeof LiveEventFilesChanged>;

export const LiveEventGrabsChanged = zod.object({
  event: zod.enum(["grabs.changed"]),
  data: GrabsChanged,
});

export type LiveEventGrabsChanged = zod.input<typeof LiveEventGrabsChanged>;
export type LiveEventGrabsChangedOutput = zod.output<typeof LiveEventGrabsChanged>;

export const LiveEventGrabsSearchChanged = zod.object({
  event: zod.enum(["grabs.search-changed"]),
  data: GrabsSearchChanged,
});

export type LiveEventGrabsSearchChanged = zod.input<typeof LiveEventGrabsSearchChanged>;
export type LiveEventGrabsSearchChangedOutput = zod.output<typeof LiveEventGrabsSearchChanged>;

export const LiveEventIdentityAccountsChanged = zod.object({
  event: zod.enum(["identity.accounts-changed"]),
  data: IdentityAccountsChanged,
});

export type LiveEventIdentityAccountsChanged = zod.input<typeof LiveEventIdentityAccountsChanged>;
export type LiveEventIdentityAccountsChangedOutput = zod.output<
  typeof LiveEventIdentityAccountsChanged
>;

export const LiveEventIdentityLinksChanged = zod.object({
  event: zod.enum(["identity.links-changed"]),
  data: IdentityLinksChanged,
});

export type LiveEventIdentityLinksChanged = zod.input<typeof LiveEventIdentityLinksChanged>;
export type LiveEventIdentityLinksChangedOutput = zod.output<typeof LiveEventIdentityLinksChanged>;

export const LiveEventIdentitySessionsChanged = zod.object({
  event: zod.enum(["identity.sessions-changed"]),
  data: IdentitySessionsChanged,
});

export type LiveEventIdentitySessionsChanged = zod.input<typeof LiveEventIdentitySessionsChanged>;
export type LiveEventIdentitySessionsChangedOutput = zod.output<
  typeof LiveEventIdentitySessionsChanged
>;

export const LiveEventIndexersChanged = zod.object({
  event: zod.enum(["indexers.changed"]),
  data: IndexersChanged,
});

export type LiveEventIndexersChanged = zod.input<typeof LiveEventIndexersChanged>;
export type LiveEventIndexersChangedOutput = zod.output<typeof LiveEventIndexersChanged>;

export const LiveEventLiveAccessChanged = zod.object({
  event: zod.enum(["live.access-changed"]),
  data: LiveAccessChanged,
});

export type LiveEventLiveAccessChanged = zod.input<typeof LiveEventLiveAccessChanged>;
export type LiveEventLiveAccessChangedOutput = zod.output<typeof LiveEventLiveAccessChanged>;

export const LiveEventLiveResync = zod.object({
  event: zod.enum(["live.resync"]),
  data: LiveResync,
});

export type LiveEventLiveResync = zod.input<typeof LiveEventLiveResync>;
export type LiveEventLiveResyncOutput = zod.output<typeof LiveEventLiveResync>;

export const LiveEventLiveSessionEnded = zod.object({
  event: zod.enum(["live.session-ended"]),
  data: LiveSessionEnded,
});

export type LiveEventLiveSessionEnded = zod.input<typeof LiveEventLiveSessionEnded>;
export type LiveEventLiveSessionEndedOutput = zod.output<typeof LiveEventLiveSessionEnded>;

export const LiveEventNodesChanged = zod.object({
  event: zod.enum(["nodes.changed"]),
  data: NodesChanged,
});

export type LiveEventNodesChanged = zod.input<typeof LiveEventNodesChanged>;
export type LiveEventNodesChangedOutput = zod.output<typeof LiveEventNodesChanged>;

export const LiveEventRootsChanged = zod.object({
  event: zod.enum(["roots.changed"]),
  data: RootsChanged,
});

export type LiveEventRootsChanged = zod.input<typeof LiveEventRootsChanged>;
export type LiveEventRootsChangedOutput = zod.output<typeof LiveEventRootsChanged>;

export const LiveEventTargetsChanged = zod.object({
  event: zod.enum(["targets.changed"]),
  data: TargetsChanged,
});

export type LiveEventTargetsChanged = zod.input<typeof LiveEventTargetsChanged>;
export type LiveEventTargetsChangedOutput = zod.output<typeof LiveEventTargetsChanged>;

export const LiveEventWorksChanged = zod.object({
  event: zod.enum(["works.changed"]),
  data: WorksWorkChanged,
});

export type LiveEventWorksChanged = zod.input<typeof LiveEventWorksChanged>;
export type LiveEventWorksChangedOutput = zod.output<typeof LiveEventWorksChanged>;

export const LiveEventWorksContentTypesChanged = zod.object({
  event: zod.enum(["works.content-types-changed"]),
  data: WorksContentTypesChanged,
});

export type LiveEventWorksContentTypesChanged = zod.input<typeof LiveEventWorksContentTypesChanged>;
export type LiveEventWorksContentTypesChangedOutput = zod.output<
  typeof LiveEventWorksContentTypesChanged
>;

export const LiveEvent = zod.discriminatedUnion("event", [
  LiveEventFilesChanged,
  LiveEventGrabsChanged,
  LiveEventGrabsSearchChanged,
  LiveEventIdentityAccountsChanged,
  LiveEventIdentityLinksChanged,
  LiveEventIdentitySessionsChanged,
  LiveEventIndexersChanged,
  LiveEventLiveAccessChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
  LiveEventNodesChanged,
  LiveEventRootsChanged,
  LiveEventTargetsChanged,
  LiveEventWorksChanged,
  LiveEventWorksContentTypesChanged,
]);

export type LiveEvent = zod.input<typeof LiveEvent>;
export type LiveEventOutput = zod.output<typeof LiveEvent>;
