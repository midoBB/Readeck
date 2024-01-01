// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// This is a postCSS plugin that provides @rules to help with
// content vertical flow.

const {Declaration, Rule} = require("postcss")

const rxUnitValue = /^([0-9.]+)([a-z%]+)$/
const varLineHeight = "--prose-line-height"

class Prose {
  constructor() {
    this.lineHeight = 1.5
  }

  /**
   *
   * @param {Declaration} node
   */
  setParams(node) {
    for (let n of node.nodes) {
      if (n.type !== "decl") {
        continue
      }
      switch (n.prop) {
        case "line-height":
          this.lineHeight = parseFloat(n.value).toFixed(4)
          break
      }
    }

    const rule = new Rule({selector: ":root"})
    rule.append(new Declaration({prop: varLineHeight, value: this.lineHeight}))
    node.replaceWith(rule)
  }

  /**
   *
   * @param {Declaration[]} nodes
   */
  getParams(nodes) {
    const res = {}
    for (let n of nodes) {
      if (n.type !== "decl") {
        continue
      }

      let value = n.value
      let unit = null
      const m = rxUnitValue.exec(n.value)
      if (m) {
        value = parseFloat(m[1]).toFixed(4)
        unit = m[2]
      }
      res[n.prop] = {value, unit, node: n}
    }

    return res
  }

  /**
   *
   * @param {Declaration} decl
   * @returns {Declaration[]}
   */
  verticalFlow(decl) {
    const params = this.getParams(decl.parent.nodes)

    if (params["font-size"] && params["font-size"].unit != "em") {
      throw params["font-size"].node.error(
        "only em unit is valid for font-size",
      )
    }

    const lineHeight = params["line-height"]?.value || this.lineHeight
    const fontSize = params["font-size"]?.value || 1
    const minLH = 0.8
    const res = [new Declaration({prop: varLineHeight, value: lineHeight})]

    // Remove params
    for (let x of ["line-height"]) {
      if (x in params) {
        params[x].node.remove()
      }
    }

    // Lines covered
    let covers = Math.ceil(fontSize / lineHeight)

    // We could use one line less
    if (covers > 1 && ((covers - 1) * lineHeight) / fontSize >= minLH) {
      covers = covers - 1
    }

    // Set line height
    let lh = (covers * lineHeight) / fontSize
    res.push(
      new Declaration({
        prop: "--prose-lh",
        value: parseFloat(lh).toFixed(4),
      }),
    )
    if (lh != lineHeight) {
      res.push(
        new Declaration({
          prop: "line-height",
          value: parseFloat(lh).toFixed(4),
        }),
      )
    }

    // Changing the line height moves the block out of the baseline, let's
    // restore it.
    if (lh > 1 && fontSize > lineHeight && lineHeight * covers != fontSize) {
      res.push(
        new Declaration({prop: "position", value: "relative"}),
        new Declaration({prop: "top", value: "0.2em"}),
      )
    }

    return res
  }

  /**
   *
   * @param {Declaration} decl
   * @returns {Declaration[]}
   */
  flowBlock(decl) {
    const res = this.verticalFlow(decl)
    const params = this.getParams(decl.parent.nodes)
    const lineHeight = params["line-height"]?.value || this.lineHeight
    const fontSize = params["font-size"]?.value || 1

    res.push(
      new Declaration({
        prop: "margin",
        value: `0 0 ${(lineHeight / fontSize).toFixed(4)}em 0`,
      }),
    )
    return res
  }
}

const prose = new Prose()

const plugin = () => {
  return {
    postcssPlugin: "prose",
    AtRule: {
      prose: (decl) => {
        prose.setParams(decl)
      },

      "vertical-flow": (decl) => {
        decl.replaceWith(...prose.verticalFlow(decl))
      },

      "flow-block": (decl) => {
        decl.replaceWith(...prose.flowBlock(decl))
      },
    },
  }
}
plugin.postcss = true

module.exports = plugin
