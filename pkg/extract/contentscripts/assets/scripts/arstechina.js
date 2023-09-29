// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "arstechnica.com"
}

exports.setConfig = function (config) {
  config.replaceStrings = [
    ['" data-src="', '"><img src="'],
    ['" data-responsive="', '" /><span data-responsive="'],
    ['<figure style="', '</span><figure data-style="'],
  ]
}
