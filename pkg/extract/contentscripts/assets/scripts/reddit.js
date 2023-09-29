// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "reddit.com"
}

exports.processMeta = function () {
  // Any reddit post provides a JSON payload when you add
  // a ".json" extension to it. Pretty neat.
  const url = new URL($.url)
  url.pathname += ".json"

  let rsp = requests.get(url)
  rsp.raiseForStatus()
  const data = rsp.json()
  const postID = data[0]?.data?.children[0]?.data?.name
  const postHint = data[0]?.data?.children[0]?.data?.post_hint

  if (postHint == "image") {
    // We have a picture!
    $.type = "photo"

    rsp = requests.get(
      "https://gateway.reddit.com/desktopapi/v1/postcomments/" + postID,
    )
    rsp.raiseForStatus()
    const post = rsp.json()

    let img = (post.posts[postID]?.media?.resolutions || []).findLast(
      () => true,
    )
    if (img) {
      $.meta["x.picture_url"] = unescapeURL(img.url)
    }

    const title = post.posts[postID]?.title
    if (title) {
      $.title = title
    }

    const author = post.posts[postID]?.author
    if (author) {
      $.authors = [author]
    }
    $.description = ""
  }
}
