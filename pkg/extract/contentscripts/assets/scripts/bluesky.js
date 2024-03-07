// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return (
    $.domain == "bsky.app" &&
    new URL($.url).pathname.match(/^\/profile\/(.+)?\/post\/([a-zA-Z0-9]+)$/)
  )
}

exports.processMeta = function () {
  const m = new URL($.url).pathname.match(
    /^\/profile\/(.+)?\/post\/([a-zA-Z0-9]+)$/,
  )
  const userName = m[1]
  const postID = m[2]

  loadThread(userName, postID)
}

function loadThread(userName, postID) {
  let rsp = requests.get(
    `https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?handle=${userName}`,
  )
  rsp.raiseForStatus()
  const did = rsp.json().did

  const postURI = `at://${did}/app.bsky.feed.post/${postID}`
  rsp = requests.get(
    `https://public.api.bsky.app/xrpc/app.bsky.feed.getPostThread?uri=${postURI}&depth=20`,
  )
  rsp.raiseForStatus()
  const data = rsp.json()
  const replies = data.thread?.replies || []

  const notes = []

  const post = getNoteData(data.thread?.post)

  notes.push(post)

  const html = notes
    .map((n) => {
      let noteHtml = `<p>${n.html}</p>`
      noteHtml = noteHtml.replace(/\n\n/g, "</p><p>")
      noteHtml = noteHtml.replace(/\n/g, "<br>")

      return `<article>${noteHtml}</article>`
    })

  $.description = ""
  $.html = `<div>${html.join("<hr>")}</div>`
  $.readability = false
}

function getNextPostFromUser(replies, userName) {
  return replies?.filter(reply =>
    reply.post.author.handle == userName
  ).reverse()[0]
}

function getNoteData(note) {
  const noteData = {
    published: note.record.createdAt,
    html: note.record.text,
  }
  return noteData
}
