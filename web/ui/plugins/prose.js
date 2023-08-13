// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// This is a postCSS plugin that provides @rules to help with
// content vertical flow.

function parseParams(params) {
  parts = params.trim().split(/[ ]+/)
  let res = []
  for (let x of parts) {
    let v = parseFloat(x)
    if (isNaN(v)) {
      throw new Error(`"${x}" is not a number`)
    }
    res.push(v)
  }

  return res
}

function roundP(n) {
  return n.toPrecision(4)
}

function verticalFlow(lineHeight, fontSize) {
  const minLH = 0.85
  const res = []

  // Lines covered
  let covers = Math.ceil(fontSize / lineHeight)

  // We could use one line less
  if (covers > 1 && ((covers - 1) * lineHeight) / fontSize >= minLH) {
    covers = covers - 1
  }

  // Set line height
  let lh = (covers * lineHeight) / fontSize
  if (lh != lineHeight) {
    // res.lineHeight = roundP(lh)
    res.push({prop: "line-height", value: roundP(lh)})
  }

  // Changing the line height moves the block out of the baseline, let's
  // restore it.
  if (fontSize > lineHeight && lineHeight * covers != fontSize) {
    res.push(
      {prop: "position", value: "relative"},
      {prop: "top", value: `${roundP(0.9 - lineHeight / fontSize)}em`},
    )
  }

  return res
}

function flowBlock(lineHeight, fontSize) {
  let res = verticalFlow(lineHeight, fontSize)
  res.push({prop: "margin", value: `0 0 ${roundP(lineHeight / fontSize)}em 0`})
  return res
}

function paddedBox(lineHeight, factor, border) {
  factor = factor || 0.5
  border = border || 0
  let p = factor * lineHeight

  let res = [{prop: "padding", value: `${p}em`}]
  if (border > 0) {
    res.push({prop: "padding", value: `calc(${p}em - ${border}px) ${p}em`})
  }

  return res
}

const plugin = () => {
  return {
    postcssPlugin: "prose",
    Declaration: {
      // vertical-flow: {line-height} {font-size}
      // This adds CSS rules to ensure a correct positionning of each line
      // of a block on the vertical grid.
      "vertical-flow": (decl) => {
        let params = []
        try {
          params = parseParams(decl.value)
          if (params.length < 2) {
            throw new Error(`only ${params.length} parameter(s)`)
          }
        } catch (e) {
          throw decl.error(`@prose-block takes 2 number parameters. (${e})`)
        }

        let res = verticalFlow(params[0], params[1])
        decl.replaceWith(...res)
      },

      // flow-block: {line-height} {font-size}
      // This adds CSS rules from @vertical-flow and adds a bottom margin
      // on the block.
      "flow-block": (decl) => {
        let params = []
        try {
          params = parseParams(decl.value)
          if (params.length < 2) {
            throw new Error(`only ${params.length} parameter(s)`)
          }
        } catch (e) {
          throw decl.error(`@prose-block takes 2 number parameters. (${e})`)
        }

        let res = flowBlock(params[0], params[1])
        decl.replaceWith(...res)
      },

      // padded-box: {line-height} {factor} {border}
      // This adds CSS rules to add a padding to a box and take into account
      // the border width.
      "padded-box": (decl) => {
        let params = []
        try {
          params = parseParams(decl.value)
          if (params.length < 3) {
            throw new Error(`only ${params.length} parameter(s)`)
          }
        } catch (e) {
          throw decl.error(`@prose-block takes 3 number parameters. (${e})`)
        }

        let res = paddedBox(params[0], params[1], params[2])
        decl.replaceWith(...res)
      },
    },
  }
}
plugin.postcss = true

module.exports = plugin
