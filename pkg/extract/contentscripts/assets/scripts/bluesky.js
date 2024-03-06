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
    `https://public.api.bsky.app/xrpc/app.bsky.feed.getPostThread?uri=${postURI}`,
  )
  rsp.raiseForStatus()
  const data = rsp.json()
  const replies = data.thread?.replies || []
  replies.reverse()

  replies
    .filter((x) => {
      return (x.replies || []).length > 0
    })
    .map((x) => {
      console.log(x)
    })

  // console.warn(JSON.stringify(data, null, "  "))
}
