// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

exports.isActive = function () {
  return $.domain == "mediapart.fr"
}

exports.setConfig = function (config) {
  config.bodySelectors.push(
    "//div[contains(concat(' ',normalize-space(@class),' '),' content-article ')]",
  )
  config.stripIdOrClass.push("cookie-consent-banner-content")
}
