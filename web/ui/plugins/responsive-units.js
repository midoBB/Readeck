// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only
"use strict"

const {Declaration} = require("postcss")
const valueParser = require("postcss-value-parser")

/**
 *
 * @param {Declaration} decl
 */
function resopnsiveUnits(decl) {
  const parsed = valueParser(decl.value)
  let modified = false
  parsed.walk((node) => {
    if (node.type != "word") {
      return
    }
    const unit = valueParser.unit(node.value)
    if (!unit || (unit.unit != "vh" && unit.unit != "vw")) {
      return
    }
    node.value = `${unit.number}d${unit.unit}`
    modified = true
  })

  if (modified) {
    // Add a new rule with the dvh value
    decl.after({prop: decl.prop, value: valueParser.stringify(parsed)})
  }
}

const plugin = () => {
  return {
    postcssPlugin: "responsive-units",
    Declaration: resopnsiveUnits,
  }
}

plugin.postcss = true
module.exports = plugin
