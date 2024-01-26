// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

const {
  parse,
  okhsl,
  interpolate,
  easingGamma,
  easingInOutSine,
  easingSmoothstep,
} = require("culori")
const {Declaration} = require("postcss")
const valueParser = require("postcss-value-parser")

const twScale = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950]

const clamp = (value) => Math.max(0, Math.min(1, value || 0))
const fixup = (value) => Math.round(clamp(value) * 255)

function getScale(
  options = {base, from, to, gamma, inoutsine, smoothstep, reversed},
) {
  options = {
    gamma: null,
    inoutsine: false,
    smoothstep: false,
    reversed: false,
    ...options,
  }

  const base = okhsl(options.base)
  let from = okhsl(options.from || "#fff")
  let to = okhsl(options.to || "#000")

  const helpers = []
  if (!!options.inoutsine) {
    helpers.push(easingInOutSine)
  }
  if (!!options.smoothstep) {
    helpers.push(easingSmoothstep)
  }
  if (options.gamma !== null) {
    helpers.push(easingGamma(options.gamma))
  }

  const inpterpolator = interpolate([...helpers, from, base, to])

  let res = twScale.map((x) => x / 1000).map(inpterpolator)

  if (!!options.reversed) {
    res.reverse()
  }

  return res
}

function setRGBValues(decl) {
  const nodes = valueParser(decl.value).nodes[0].nodes.filter(
    (n) => n.type == "word",
  )
  const c = parse(nodes[0].value)
  decl.value = `${fixup(c.r)} ${fixup(c.g)} ${fixup(c.b)}`
}

function setColorScale(decl) {
  const nodes = valueParser(decl.value).nodes[0].nodes.filter(
    (n) => n.type == "word",
  )

  const params = {}
  for (let i = 0; i < nodes.length; i++) {
    if (i == 0) {
      params.base = nodes[i].value
      continue
    }

    switch (nodes[i].value) {
      case "reversed":
        params.reversed = true
        break
      case "inoutsine":
        params.inoutsine = true
        break
      case "smoothstep":
        params.smoothstep = true
        break
      case "gamma":
        params.gamma = nodes[i + 1].value
        i++
        break
    }
  }
  decl.replaceWith(
    ...getScale(params).map((c, i) => {
      return new Declaration({
        prop: `${decl.prop}-${twScale[i]}`,
        value: `${fixup(c.r)} ${fixup(c.g)} ${fixup(c.b)}`,
      })
    }),
  )
}

const plugin = () => {
  return {
    postcssPlugin: "palette",
    Declaration(decl) {
      if (!decl.prop.startsWith("--")) {
        return
      }
      if (decl.value.startsWith("color-scale(")) {
        setColorScale(decl)
      }
      if (decl.value.startsWith("rgb-values(")) {
        setRGBValues(decl)
      }
    },
  }
}
plugin.postcss = true

module.exports = plugin
