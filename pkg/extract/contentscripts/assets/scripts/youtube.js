// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "youtube.com" && $.meta["schema.identifier"].length > 0
}

exports.processMeta = function () {
  const videoID = $.meta["schema.identifier"][0]

  const info = getVideoInfo(videoID)

  // Get more information
  const lengthSeconds = info.videoDetails?.lengthSeconds
  if (lengthSeconds) {
    $.meta["x.duration"] = lengthSeconds
  }

  // Get transcript
  const transcript = getTranscript(info)
  if (transcript) {
    $.html = `<section id="main"><p>${transcript.join("<br>\n")}</p></section>`
    // we must force readability here for it to pick up the content
    // (it normally won't with a video)
    $.readability = true
  }
}

function getVideoInfo(videoID) {
  let rsp = requests.post(
    "https://youtubei.googleapis.com/youtubei/v1/player",
    JSON.stringify(
      {
        context: {
          client: {
            hl: "en",
            clientName: "WEB",
            clientVersion: "2.20210721.00.00",
            mainAppWebInfo: {
              graftUrl: "/watch?v=" + videoID,
            },
          },
        },
        videoId: videoID,
      },
      {
        "Content-Type": "application/json",
      },
    ),
  )
  rsp.raiseForStatus()
  return rsp.json()
}

function getTranscript(info) {
  let captions =
    info.captions?.playerCaptionsTracklistRenderer?.captionTracks || []

  captions.sort((a, b) => {
    if (b.kind == "asr" && a.kind != "asr") {
      return -1
    } else if (a.kind == "asr" && b.kind != "asr") {
      return 1
    }
    return 0
  })

  const track = (captions || []).find((x) => true)
  if (!track) {
    return
  }

  console.debug("found transcript", { track })

  const rsp = requests.get(track.baseUrl)
  rsp.raiseForStatus()

  return (decodeXML(rsp.text()).transcript?.text || [])
    .map((x) => {
      return x["#text"]
    })
    .filter((x) => x)
}
