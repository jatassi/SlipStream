import {
  ArrowReloadHorizontalIcon,
  CpuIcon,
  PackageMovingIcon,
  PlayIcon,
} from "@hugeicons/core-free-icons";

import type {
  PlaybackBlocked as Blocked,
  PlaybackBlockedDimension as Dimension,
  PlaybackEvidenceFact as FactKind,
  MediaserversSessionView as Session,
  MediaserversSessionViewDecision as Decision,
} from "../../api/gen/model";
import type { StatusMark, TermsFor } from "../../ui/state-marks";

/** What each Jellyfin and Emby TranscodeReason means, as it reads after "because". */
export const reasonLines: Readonly<Record<string, string>> = {
  ContainerNotSupported: "the player can't play its container",
  VideoCodecNotSupported: "the player can't play its video codec",
  AudioCodecNotSupported: "the player can't play its audio codec",
  SubtitleCodecNotSupported: "the player can't show its subtitles",
  SubtitleContentOptionsEnabled: "the player's subtitle style options are on",
  AudioIsExternal: "its audio is in a separate file",
  ExternalAudioNotSupported: "its audio is in a separate file",
  SecondaryAudioNotSupported: "the player can't play a second audio track",
  AudioDelayNotSupported: "an audio delay is set on the player",
  StreamCountExceedsLimit: "it has more streams than the player allows",
  VideoProfileNotSupported: "the player can't play its video profile",
  VideoLevelNotSupported: "the player can't play its video level",
  VideoCodecTagNotSupported: "the player can't play its video codec tag",
  RefFramesNotSupported: "the player can't play its reference frames",
  VideoRangeTypeNotSupported: "the player can't show its HDR format",
  VideoRangeNotSupported: "the player can't show its HDR format",
  VideoResolutionNotSupported: "the player can't play its resolution",
  VideoBitDepthNotSupported: "the player can't play its video bit depth",
  VideoFramerateNotSupported: "the player can't play its frame rate",
  VideoRotationNotSupported: "the player can't play rotated video",
  AnamorphicVideoNotSupported: "the player can't play anamorphic video",
  InterlacedVideoNotSupported: "the player can't play interlaced video",
  AudioChannelsNotSupported: "the player can't play its audio channels",
  AudioProfileNotSupported: "the player can't play its audio profile",
  AudioSampleRateNotSupported: "the player can't play its audio sample rate",
  AudioBitDepthNotSupported: "the player can't play its audio bit depth",
  ContainerBitrateExceedsLimit: "its bitrate is above the player's limit",
  VideoBitrateNotSupported: "its video bitrate is above the player's limit",
  AudioBitrateNotSupported: "its audio bitrate is above the player's limit",
  UnknownVideoStreamInfo: "the server can't read its video stream",
  UnknownAudioStreamInfo: "the server can't read its audio stream",
  DirectPlayError: "direct play failed on the player",
};

type Shown = Decision | "hardware_transcode";

const decisionTerms = {
  direct_play: {
    word: "Direct play",
    icon: PlayIcon,
    tone: "neutral",
    definition: "The player plays the original file.",
  },
  direct_stream: {
    word: "Direct stream",
    icon: PackageMovingIcon,
    tone: "neutral",
    definition:
      "The server copies the file's streams into a new container without transcoding them.",
  },
  transcode: {
    word: "Transcode",
    icon: ArrowReloadHorizontalIcon,
    tone: "neutral",
    definition:
      "The server re-encodes the video or audio into a format the player can play, or burns in the subtitles.",
  },
  hardware_transcode: {
    word: "Hardware transcode",
    icon: CpuIcon,
    tone: "neutral",
    definition:
      "The server re-encodes the file on its graphics hardware into a format the player can play.",
  },
} as const satisfies TermsFor<Shown, "neutral">;

/** The standard name for what the server does with the file, with its definition. */
export function decisionTerm(play: Pick<Session, "decision" | "hardware">): StatusMark {
  return decisionTerms[
    play.decision === "transcode" && play.hardware ? "hardware_transcode" : play.decision
  ];
}

const list = new Intl.ListFormat("en", { type: "conjunction" });

function transcoded(session: Session): string {
  const video = session.video === "transcode" || session.subtitles === "burned";
  const audio = session.audio === "transcode";
  const streams = video && audio ? "the video and audio" : video ? "the video" : "the audio";
  const hardware = session.hardware ? " on its graphics hardware" : "";
  const burned = audio ? " and burns in its subtitles" : " to burn in its subtitles";
  const burn = session.subtitles === "burned" ? burned : "";
  const reasons = session.reasons.map((reason) => reasonLines[reason] ?? reason);
  const why = reasons.length === 0 ? "" : ` because ${list.format(reasons)}`;
  return `transcodes ${streams}${hardware}${burn}${why}`;
}

/** What the server does with a live session's streams and why, or undefined for a direct play. */
export function nowPlayingLine(server: string, session: Session): string | undefined {
  if (session.decision === "direct_play") {
    return undefined;
  }
  if (session.decision === "direct_stream") {
    return `${server} copies the file's streams into a new container without transcoding them.`;
  }
  return `${server} ${transcoded(session)}.`;
}

const hdrWords: Readonly<Record<string, string>> = { DV: "Dolby Vision" };

/** An HDR format as the Target editor's filter names it in a sentence. */
export const hdrWord = (value: string): string => hdrWords[value] ?? value;

const megabits = new Intl.NumberFormat("en", { maximumFractionDigits: 1 });

const bitrateWords = (bits: string) => `${megabits.format(Number(bits) / 1_000_000)} Mbps`;

/** A dimension a Target can choose, whose envelope a player proves value by value. */
type Choosable = {
  readonly lead: (value: string) => string;
  readonly unproven: (value: string) => string;
  readonly side: "video" | "audio";
};

/** A dimension no Target fixes, and what the Admin can do instead. */
type Fixed = {
  readonly lead: (value: string) => string;
  readonly follow: (server: string) => string;
};

const noTarget = "No Target fixes this";

/**
 * Each dimension's words after "because", and for one no Target fixes, what
 * follows. A dimension a Target can choose words what a player hasn't proven
 * on it instead.
 */
export const dimensionWords: { readonly [D in Dimension]: Choosable | Fixed } = {
  resolution: {
    lead: (value) => `the player can't play ${value} video`,
    unproven: (value) => `video below ${value}`,
    side: "video",
  },
  video_codec: {
    lead: (value) => `the player can't play ${value} video`,
    unproven: () => "video in another codec",
    side: "video",
  },
  hdr: {
    lead: (value) => `the player can't show ${hdrWord(value)}`,
    unproven: (value) => `video without ${hdrWord(value)}`,
    side: "video",
  },
  audio_codec: {
    lead: (value) => `the player can't play ${value} audio`,
    unproven: () => "audio in another codec",
    side: "audio",
  },
  channels: {
    lead: (value) => `the player can't play ${value} audio`,
    unproven: (value) => `audio with fewer channels than ${value}`,
    side: "audio",
  },
  container: {
    lead: (value) => `the player can't play the ${value} container`,
    follow: () => `${noTarget}, because a Target can't choose a file's container.`,
  },
  subtitles: {
    lead: () => "the player can't show the file's subtitles",
    follow: () => `${noTarget}. Turn subtitles off or choose text subtitles on the player.`,
  },
  audio_track: {
    lead: (reason) => reasonLines[reason] ?? reason,
    follow: () => `${noTarget}. Choose another audio track on the player.`,
  },
  stream_count: {
    lead: () => "the file has more streams than the player allows",
    follow: () => `${noTarget}, because a Target can't choose how many streams a file has.`,
  },
  video_profile: {
    lead: () => "the player can't play the file's video profile or level",
    follow: () => `${noTarget}, because a Target can't choose a file's video profile.`,
  },
  bit_depth: {
    lead: (value) => `the player can't play ${value}-bit video`,
    follow: () => `${noTarget}, because a Target can't choose a file's bit depth.`,
  },
  frame_rate: {
    lead: () => "the player can't play the file's frame rate",
    follow: () => `${noTarget}, because a Target can't choose a file's frame rate.`,
  },
  video_shape: {
    lead: (reason) => reasonLines[reason] ?? reason,
    follow: () => `${noTarget}, because a Target can't choose how a file's video is encoded.`,
  },
  audio_format: {
    lead: (reason) => reasonLines[reason] ?? reason,
    follow: () => `${noTarget}, because a Target can't choose a file's audio format.`,
  },
  bitrate: {
    lead: () => "the file's bitrate is above the player's limit",
    follow: () =>
      `${noTarget}. Check the player's quality setting and the server's limit for streams outside your network.`,
  },
  unreadable: {
    lead: (stream) => `the server can't read the file's ${stream} stream`,
    follow: (server) => `${noTarget}. Refresh the item's metadata on ${server}.`,
  },
  player_error: {
    lead: () => "direct play failed on the player",
    follow: () => `${noTarget}. Update the player app or check its playback settings.`,
  },
  unknown: {
    lead: () => "the server gave no reason",
    follow: () => "Check the player's quality setting.",
  },
};

const leadWords = (blocked: Blocked) => dimensionWords[blocked.dimension].lead(blocked.value);

const playerCant = "the player can't ";

/** The causes a line names, joined as "the player can't show Dolby Vision or HDR10 or play TrueHD audio". */
export function causeWords(cause: readonly Blocked[]): string {
  const clauses = cause.map((blocked) => leadWords(blocked));
  return clauses
    .map((words, i) => {
      const before = clauses[i - 1];
      if (before === undefined || !before.startsWith(playerCant) || !words.startsWith(playerCant)) {
        return words;
      }
      const rest = words.slice(playerCant.length);
      const [verb = ""] = rest.split(" ", 1);
      return before.startsWith(`${playerCant}${verb} `) ? rest.slice(verb.length + 1) : rest;
    })
    .join(" or ");
}

/** A tracked fact's value as a sentence names it after "its", such as "MP4 container". */
export const factWords: { readonly [F in FactKind]: (value: string) => string } = {
  container: (value) => `${value} container`,
  bitrate: (value) => `bitrate of ${bitrateWords(value)}`,
  resolution: (value) => `${value} video`,
  video_codec: (value) => `${value} video`,
  video_profile: (value) => `${value} video profile`,
  bit_depth: (value) => `${value}-bit video`,
  hdr: (value) => hdrWord(value),
  dolby_vision: (value) => `Dolby Vision profile ${value}`,
  audio_codec: (value) => `${value} audio`,
  channels: (value) => `${value} audio`,
  subtitle_codec: (value) => `${value} subtitles`,
  subtitle_image: (value) => (value === "true" ? "image subtitles" : "text subtitles"),
};
