// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "vimeo.com"
}

exports.setConfig = function (config) {
  config.singlePageLinkSelectors = []
}

// Vimeo big thumbnail includes a play button.
// The actual picture URL is given by one of the URL
// query parameters (src0)
exports.processMeta = function () {
  const twPicture = $.meta["graph.image"]
  if (twPicture.length == 0) {
    return
  }

  const url = new URL(twPicture[0])
  const img = url.searchParams.get("src0")
  if (img !== null) {
    $.meta["x.picture_url"] = img
  }
}
