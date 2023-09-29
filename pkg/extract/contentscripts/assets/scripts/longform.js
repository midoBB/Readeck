// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "longform.org"
}

exports.setConfig = function (config) {
  config.singlePageLinkSelectors.push("//a[@href][@class='post__link']/@href")
}
