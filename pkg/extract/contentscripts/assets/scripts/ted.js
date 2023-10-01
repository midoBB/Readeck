// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "ted.com" && new URL($.url).pathname.match(/^\/talks\//)
}

exports.processMeta = function () {
  const videoId = getVideoId()
  const transcript = getTranscript(videoId)
  if (transcript) {
    console.debug("found transcript", { videoId })
    $.html = `<section id="main">${transcript
      .map((p) => {
        return `<p>${p}</p>`
      })
      .join("\n")}</section>`
    $.readability = true
  }
}

function getVideoId() {
  return new URL($.url).pathname.replace(/\/talks\/(.+?)(\/.*)?$/, "$1")
}

function getTranscript(videoId) {
  const rsp = requests.post(
    "https://www.ted.com/graphql",
    JSON.stringify({
      operationName: "Transcript",
      variables: {
        id: videoId,
        language: "en",
      },
      query: graphqlQuery,
    }),
    {
      "Content-Type": "application/json",
    },
  )
  rsp.raiseForStatus()

  // TED transcript is a simple format with a very good structured result.
  return (rsp.json().data?.translation?.paragraphs || []).map((p) => {
    return (p.cues || [])
      .map((cue) => {
        return cue.text
      })
      .join(" ")
  })
}

const graphqlQuery = `query Transcript($id: ID!, $language: String!) {
  translation(videoId: $id, language: $language) {
    id
    language {
      id
      endonym
      englishName
      internalLanguageCode
      rtl
      __typename
    }
    reviewer {
      id
      uri
      avatar {
        url
        generatedUrl(type: SVG)
        __typename
      }
      name {
        full
        __typename
      }
      __typename
    }
    translator {
      id
      uri
      avatar {
        url
        generatedUrl(type: SVG)
        __typename
      }
      name {
        full
        __typename
      }
      __typename
    }
    paragraphs {
      cues {
        text
        time
        __typename
      }
      __typename
    }
    __typename
  }
  video(id: $id, language: $language) {
    id
    talkExtras {
      footnotes {
        author
        annotation
        date
        linkUrl
        source
        text
        timecode
        title
        category
        __typename
      }
      __typename
    }
    __typename
  }
}
`
