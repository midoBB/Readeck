// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "bostonglobe.com"
}

exports.setConfig = function (config) {
  config.bodySelectors = ["//article"]
  config.stripIdOrClass = ["tagline", "tagline_hr"]
}
