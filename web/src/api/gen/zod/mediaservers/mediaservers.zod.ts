import * as zod from "zod";

import {
  HoldersOutputBody,
  KindsOutputBody,
  LinkUserInputBody,
  MakeAccountInputBody,
  MappingsOutputBody,
  MediaserversAccessList,
  MediaserversAccountView,
  MediaserversActed,
  MediaserversChanges,
  MediaserversDraftBody,
  MediaserversLibraries,
  MediaserversMappingsBody,
  MediaserversMemberPin,
  MediaserversMemberPinState,
  MediaserversMyAccess,
  MediaserversMyAccessRow,
  MediaserversPatchBody,
  MediaserversPinBody,
  MediaserversPlays,
  MediaserversPlexPin,
  MediaserversPlexPinState,
  MediaserversServerView,
  MediaserversSessions,
  MediaserversSettings,
  MediaserversSignInBody,
  MediaserversSignedIn,
  MediaserversTested,
  OfferInputBody,
  PlaybackReport,
  PlexServersOutputBody,
  ServerUsersOutputBody,
  ServersOutputBody,
  SettingsInputBody,
  ShareAccessInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Media Servers the Member was offered, with what each needs next and how to watch on it
 */
export const ListMyMediaServerAccessResponse = MediaserversMyAccess;

/**
 * @summary Forget the Plex account the Member signed in to Plex as, while no server is shared with it yet
 */
export const UseAnotherPlexAccountResponse = zod.void();

/**
 * @summary Start the Member's Plex sign-in and get the plex.tv page they sign in on
 */
export const StartMemberPlexSignInResponse = MediaserversMemberPin;

/**
 * @summary Get whether the Member's Plex sign-in is still waiting, has expired, signed in or was refused
 */
export const GetMemberPlexSignInParams = zod.object({
  id: zod.int(),
});

export const GetMemberPlexSignInResponse = MediaserversMemberPinState;

/**
 * @summary Make the Member's own user on a Jellyfin or Emby server they were offered
 */
export const MakeMediaServerAccountParams = zod.object({
  accessId: zod.uuid(),
});

export const MakeMediaServerAccountBody = MakeAccountInputBody;

export const MakeMediaServerAccountResponse = MediaserversMyAccessRow;

/**
 * @summary Check whether the Member's Plex share is accepted, accepting it when SlipStream can
 */
export const CheckMyMediaServerAccessParams = zod.object({
  accessId: zod.uuid(),
});

export const CheckMyMediaServerAccessResponse = MediaserversMyAccessRow;

/**
 * @summary Offer a Member access to a Media Server, or offer it again after the access was removed
 */
export const OfferMediaServerParams = zod.object({
  id: zod.uuid().describe("The Member's Account"),
  serverId: zod.uuid(),
});

export const OfferMediaServerBody = OfferInputBody;

export const OfferMediaServerResponse = MediaserversActed;

/**
 * @summary List the Media Servers SlipStream keeps up to date, each with its state
 */
export const ListMediaServersResponse = ServersOutputBody;

/**
 * @summary Add a Media Server once it responds from its Connect from Node and accepts SlipStream's sign-in
 */
export const AddMediaServerBody = MediaserversDraftBody;

export const AddMediaServerResponse = MediaserversServerView;

/**
 * @summary List Members' access to Media Servers, the rows waiting on an Admin first
 */
export const ListMediaServerAccessQueryParams = zod.object({
  account: zod
    .uuid()
    .exactOptional()
    .describe("An Account id: only that Member's rows, and the servers they may be offered"),
});

export const ListMediaServerAccessResponse = MediaserversAccessList;

/**
 * @summary Remove a Member's access to a Media Server, on the server or plex.tv too
 */
export const RemoveMediaServerAccessParams = zod.object({
  id: zod.uuid(),
});

export const RemoveMediaServerAccessResponse = MediaserversActed;

/**
 * @summary Check whether a Member accepted a Plex share
 */
export const CheckMediaServerAccessParams = zod.object({
  id: zod.uuid(),
});

export const CheckMediaServerAccessResponse = MediaserversActed;

/**
 * @summary Link a Member to an existing user on a Jellyfin or Emby server
 */
export const LinkMediaServerUserParams = zod.object({
  id: zod.uuid(),
});

export const LinkMediaServerUserBody = LinkUserInputBody;

export const LinkMediaServerUserResponse = MediaserversActed;

/**
 * @summary Record that an Admin shared a Plex server with a Member in Plex itself
 */
export const MarkMediaServerAccessAddedParams = zod.object({
  id: zod.uuid(),
});

export const MarkMediaServerAccessAddedResponse = MediaserversActed;

/**
 * @summary Share a Plex server with a Member who signed in to Plex, as a friend or in the Plex Home
 */
export const ShareMediaServerAccessParams = zod.object({
  id: zod.uuid(),
});

export const ShareMediaServerAccessBody = ShareAccessInputBody;

export const ShareMediaServerAccessResponse = MediaserversActed;

/**
 * @summary List the kinds of Media Server SlipStream supports, with the sign-in each takes and its oldest supported version
 */
export const ListMediaServerKindsResponse = KindsOutputBody;

/**
 * @summary Sign out of Plex, which also removes every Plex server
 */
export const SignOutOfPlexResponse = zod.void();

/**
 * @summary Get the Plex account SlipStream is signed in to and when its sign-in expires
 */
export const GetPlexAccountResponse = MediaserversAccountView;

/**
 * @summary Start a Plex sign-in and get the plex.tv page the Admin signs in on
 */
export const StartPlexSignInBody = MediaserversPinBody;

export const StartPlexSignInResponse = MediaserversPlexPin;

/**
 * @summary Get whether a Plex sign-in is still waiting, has expired or signed in
 */
export const GetPlexSignInParams = zod.object({
  id: zod.int(),
});

export const GetPlexSignInResponse = MediaserversPlexPinState;

/**
 * @summary Sign SlipStream in to the different Plex account a Plex sign-in names, which removes every Plex server and Members' access to them
 */
export const ReplacePlexAccountParams = zod.object({
  id: zod.int(),
});

export const ReplacePlexAccountResponse = MediaserversAccountView;

/**
 * @summary List the Plex account's servers and which Nodes reach each of their connections
 */
export const ListPlexServersResponse = PlexServersOutputBody;

/**
 * @summary Get the Media Servers settings
 */
export const GetMediaServerSettingsResponse = MediaserversSettings;

/**
 * @summary Change the Media Servers settings by key, all or none
 */
export const UpdateMediaServerSettingsBody = SettingsInputBody;

export const UpdateMediaServerSettingsResponse = MediaserversSettings;

/**
 * @summary Sign in to a Jellyfin or Emby server as an administrator once, creating an API key named SlipStream for a draft server
 */
export const SignInToMediaServerBody = MediaserversSignInBody;

export const SignInToMediaServerResponse = MediaserversSignedIn;

/**
 * @summary Remove a Media Server, which also deletes its queued changes, its path index and its play history
 */
export const RemoveMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const RemoveMediaServerResponse = zod.void();

/**
 * @summary Get a Media Server with its state
 */
export const GetMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const GetMediaServerResponse = MediaserversServerView;

/**
 * @summary Change a Media Server, testing it again when the change is to its address, Connect from Node or API key
 */
export const UpdateMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const UpdateMediaServerBody = MediaserversPatchBody;

export const UpdateMediaServerResponse = MediaserversServerView;

/**
 * @summary List a Media Server's recent changes, newest first, with their results and what each move couldn't carry
 */
export const ListMediaServerChangesParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerChangesQueryParams = zod.object({
  cursor: zod
    .uuid()
    .exactOptional()
    .describe("The next member of the previous page; absent for the newest"),
});

export const ListMediaServerChangesResponse = MediaserversChanges;

/**
 * @summary Confirm that the server now responding at a Media Server's address is that Media Server
 */
export const ConfirmMediaServerIdentityParams = zod.object({
  id: zod.uuid(),
});

export const ConfirmMediaServerIdentityResponse = MediaserversServerView;

/**
 * @summary List every Node that fetched a Media Server's credential
 */
export const ListMediaServerCredentialHoldersParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerCredentialHoldersResponse = HoldersOutputBody;

/**
 * @summary List a Media Server's library folders and every library root's Path mapping, pre-filled where none is saved
 */
export const ListMediaServerLibrariesParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerLibrariesResponse = MediaserversLibraries;

/**
 * @summary Save a Media Server's Path mappings once each shown folder sits inside one of its library folders
 */
export const SaveMediaServerMappingsParams = zod.object({
  id: zod.uuid(),
});

export const SaveMediaServerMappingsBody = MediaserversMappingsBody;

export const SaveMediaServerMappingsResponse = MappingsOutputBody;

/**
 * @summary Get a Media Server's Playback report: why each player transcoded the files it played, and which file or Target would avoid it
 */
export const GetMediaServerPlaybackParams = zod.object({
  id: zod.uuid(),
});

export const GetMediaServerPlaybackResponse = PlaybackReport;

/**
 * @summary List a Media Server's plays in the play history, newest first, for one player or all
 */
export const ListMediaServerPlaysParams = zod.object({
  id: zod.uuid(),
});

export const listMediaServerPlaysQueryPlayerMax = 256;

export const ListMediaServerPlaysQueryParams = zod.object({
  player: zod
    .string()
    .max(listMediaServerPlaysQueryPlayerMax)
    .exactOptional()
    .describe("A player's key; absent for every player"),
  cursor: zod
    .uuid()
    .exactOptional()
    .describe("The next member of the previous page; absent for the newest"),
});

export const ListMediaServerPlaysResponse = MediaserversPlays;

/**
 * @summary Queue a whole library refresh of every library a Media Server's confirmed Path mappings lead into
 */
export const RefreshMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const RefreshMediaServerResponse = zod.void();

/**
 * @summary List what plays on a Media Server now, as its Connect from Node last saw it
 */
export const ListMediaServerSessionsParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerSessionsResponse = MediaserversSessions;

/**
 * @summary Test a Media Server from its Connect from Node at its address and from the Controller at its fallback address
 */
export const TestMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const TestMediaServerResponse = MediaserversTested;

/**
 * @summary List the users of a Jellyfin or Emby server that an Admin may link to a Member
 */
export const ListMediaServerUsersParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerUsersResponse = ServerUsersOutputBody;
