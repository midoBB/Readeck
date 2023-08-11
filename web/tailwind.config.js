const colors = require("tailwindcss/colors")

const palette = {
  blue: {
    50: "#daeff7",
    100: "#c9e7f3",
    200: "#a7d8ec",
    300: "#86cae5",
    400: "#64bbdd",
    500: "#43acd6",
    600: "#288fb9",
    700: "#1e6c8b",
    800: "#14485d",
    900: "#0a242e",
    950: "#051217",
  },
  orange: {
    50: "#fffbeb",
    100: "#fef3c7",
    200: "#fde68a",
    300: "#fcd34d",
    400: "#fbbf24",
    500: "#f59e0b",
    600: "#d97706",
    700: "#b45309",
    800: "#92400e",
    900: "#78350f",
    950: "#451a03",
  },
  red: {
    50: "#fef4f6",
    100: "#fce2e7",
    200: "#f8bdc8",
    300: "#f498a9",
    400: "#f0738a",
    500: "#ed4e6b",
    600: "#e9294c",
    700: "#d31638",
    800: "#a0112b",
    900: "#6d0b1d",
    950: "#540916",
  },
  gray: {
    50: "#fbfafa",
    100: "#f1efed",
    200: "#ded7d2",
    300: "#cac0b8",
    400: "#b2a398",
    500: "#9a8778",
    600: "#7d6b5d",
    700: "#5d4f45",
    800: "#3c342d",
    900: "#1c1815",
    950: "#0c0b09",
  },
}

// prettier-ignore
var config = {
  content: [
    "src/**/*.js",
    "../assets/templates/**/*.jet.html",
  ],
  theme: {
    colors: {
      transparent: "transparent",
      current: "currentColor",
      black: colors.black,
      white: colors.white,
      gray: palette.gray,
      red: palette.red,
      green: colors.lime,
      blue: palette.blue,
      yellow: palette.orange,
      primary: {
        light: palette.blue[300],
        DEFAULT: palette.blue[600],
        dark: palette.blue[800],
      },
    },
    fontFamily: {
      sans: [
        "public sans", "sans-serif",
        "Apple Color Emoji", "Segoe UI Emoji",
        "Segoe UI Symbol", "Noto Color Emoji",
      ],
      serif: [
        "lora", "serif",
        "Apple Color Emoji", "Segoe UI Emoji",
        "Segoe UI Symbol", "Noto Color Emoji",
      ],
    },
    screens: {
      "sm": "640px",
      "md": "814px",
      "lg": "1024px",
      "xl": "1280px",
    },
    extend: {
      boxShadow: {
        "sidebar-l": "5px 0 10px -5px rgba(0, 0, 0, 0.1)",
        "panel-t": "0 4px 6px 6px rgba(0, 0, 0, 0.1)",
      },
      contrast: {
        "105": "1.05",
      },
      fontSize: {
        "h1": "2.5rem",
        "h2": "2rem",
        "h3": "1.5rem",
      },
      gridTemplateColumns: {
        "bk-tools": "2fr auto auto",
        "cards": "repeat(auto-fill, minmax(12rem, 1fr))",
      },
      height: {
        "screen": "100vh",
        "max-content": "max-content",
        "topnav": "4rem",
      },
      padding: {
        "16/9": "56.25%",
      },
      spacing: {
        "topnav": "4rem",
        ...[...Array(26).keys()].reduce((acc, x) => {
          acc[`col-${x+1}`] = `${(x+1) * 3.5}rem`
          return acc
        }, {}),
      },
      width: {
      },
      maxWidth: {
        ...[...Array(26).keys()].reduce((acc, x) => {
          acc[`col-${x+1}`] = `${(x+1) * 3.5}rem`
          return acc
        }, {}),
      },
    },
  },
  plugins: [
    require("./ui/plugins/interactions"),
    require("./ui/plugins/forms"),
  ],
}

module.exports = config
