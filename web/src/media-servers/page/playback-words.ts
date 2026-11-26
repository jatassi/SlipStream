import {
  ArrowReloadHorizontalIcon,
  CpuIcon,
  PackageMovingIcon,
  PlayIcon,
} from "@hugeicons/core-free-icons";

import type {
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
      "The server converts the video or audio into a format the player can play, or burns in the subtitles.",
  },
  hardware_transcode: {
    word: "Hardware transcode",
    icon: CpuIcon,
    tone: "neutral",
    definition:
      "The server converts the file on its graphics hardware into a format the player can play.",
  },
} as const satisfies TermsFor<Shown, "neutral">;

/** The standard name for what the server does with the file, with its definition. */
export function decisionTerm(session: Session): StatusMark {
  return decisionTerms[
    session.decision === "transcode" && session.hardware ? "hardware_transcode" : session.decision
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
