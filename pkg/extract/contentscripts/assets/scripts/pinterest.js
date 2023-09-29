// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return (
    $.domain == "pinterest.com" && new URL($.url).pathname.match(/^\/pin\//)
  )
}

// A pin is an image (and a link to somewhere). Set it as a photo
// and call it a day.
exports.processMeta = function () {
  let img = $.meta["graph.image"]
  if (img.length > 0) {
    img = unescapeURL(img[0])
    $.type = "photo"
    $.meta["x.picture_url"] = img
  }
}
