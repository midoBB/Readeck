// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

const plugin = require("tailwindcss/plugin")

module.exports = plugin(function ({addVariant, config, e}) {
  const prefixClass = function (className) {
    const prefix = config("prefix")
    const getPrefix = typeof prefix === "function" ? prefix : () => prefix
    return `${getPrefix(`.${className}`)}${className}`
  }

  addVariant("js", ({modifySelectors, separator}) => {
    modifySelectors(({className}) => {
      return `body.js .${e(`js${separator}${className}`)}`
    })
  })

  addVariant("no-js", ({modifySelectors, separator}) => {
    modifySelectors(({className}) => {
      return `body.no-js .${e(`no-js${separator}${className}`)}`
    })
  })

  addVariant("data-current", ({modifySelectors, separator}) => {
    modifySelectors(({className}) => {
      return `.${e(
        `data-current${separator}${className}`,
      )}[data-current='true']`
    })
  })

  addVariant("hf", ["&:hover", "&:focus"])
  addVariant("hfw", ["&:hover", "&:focus", "&:focus-within"])
  addVariant("group-hf", [":merge(.group):hover &", ":merge(.group):focus &"])
  addVariant("group-fw", [":merge(.group):focus-within &"])
  addVariant("group-hfw", [
    ":merge(.group):hover &",
    ":merge(.group):focus &",
    ":merge(.group):focus-within &",
  ])
})
