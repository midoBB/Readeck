// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// This keeps only woff2 src in @font-face rules.

"use strict"

const {Node} = require("postcss")
const valueParser = require("postcss-value-parser")

/**
 *
 * @param {Node} node
 */
function filterFontTypes(node) {
  if (node.prop != "src") {
    return
  }

  const sources = node.value.split(/\s*,\s*/).filter((s) => {
    const parsed = valueParser(s)
    let isWoff2 = false
    parsed.walk((n) => {
      if (n.type == "function" && n.value == "format") {
        isWoff2 =
          n.nodes.find((n) => {
            return n.type == "string" && n.value.startsWith("woff2")
          }) !== undefined
      }
    })

    return isWoff2
  })

  node.value = sources.join(", ")
}

const plugin = () => {
  return {
    postcssPlugin: "trim-fonts",
    // Declaration: resopnsiveUnits,
    AtRule: {
      "font-face": (r) => {
        r.walk(filterFontTypes)
      },
    },
  }
}

plugin.postcss = true
module.exports = plugin
