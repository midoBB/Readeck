const colors = require("tailwindcss/colors")

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
      gray: colors.stone,
      red: colors.red,
      green: colors.lime,
      blue: colors.sky,
      yellow: colors.amber,
      primary: {
        light: colors.sky[300],
        DEFAULT: colors.sky[600],
        dark: colors.sky[800],
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
      "sm": {min: "640px"},
      "md": {min: "769px"},
      "lg": {min: "1024px"},
      "+md": {max: "768px"},
      "+sm": {max: "639px"},
      "print": {raw: "print"},
    },
    extend: {
      boxShadow: {
        "sidebar": "5px 0 10px -5px rgba(0, 0, 0, 0.1)",
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
        "max-content": "max-content",
      },
      padding: {
        "16/9": "56.25%",
      },
      width: {
        "md": "28rem",
      },
      maxWidth: {
        "view": "72rem",
      },
    },
  },
  plugins: [
    require("./ui/plugins/interactions"),
    require("./ui/plugins/forms"),
  ],
}

module.exports = config
