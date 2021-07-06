const colors = require("tailwindcss/colors")

var config = {
  purge: {
    content:[
      "src/**/*.js",
      "../assets/templates/**/*.jet.html",
    ],
    options: {
      safelist: [],
    },
  },
  darkMode: false, // or 'media' or 'class'
  theme: {
    colors: {
      transparent: "transparent",
      current: "currentColor",
      black: colors.black,
      white: colors.white,
      gray: colors.warmGray,
      red: colors.red,
      green: colors.lime,
      blue: colors.lightBlue,
      yellow: colors.amber,
      primary: {
        light: colors.lightBlue[300],
        DEFAULT: colors.lightBlue[600],
        dark: colors.lightBlue[800],
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
  variants: {
    extend: {
      backgroundColor: [
        "data-current",
        "group-hf",
        "group-focus-within",
        "hf",
      ],
      backgroundOpacity: [
        "data-current",
        "group-hf",
        "hf",
      ],
      borderColor: [
        "data-current",
        "group-hf",
        "hf",
      ],
      borderOpacity: [
        "group-hf",
        "hf",
      ],
      boxShadow: [
        "group-hf",
        "hf",
      ],
      brightness: [
        "group-hf",
        "group-hover",
        "group-focus-within",
        "hf",
      ],
      contrast: [
        "group-hf",
        "group-hover",
        "group-focus-within",
        "hf",
      ],
      display: [
        "js",
        "no-js",
      ],
      filter: [
        "group-hf",
        "group-hover",
        "group-focus-within",
        "hf",
      ],
      fontWeight: [
        "data-current",
      ],
      opacity: [
        "group-hf",
        "group-hover",
        "group-focus-within",
        "hf",
      ],
      ringWidth: [
        "hf",
      ],
      ringColor: [
        "hf",
      ],
      textColor: [
        "data-current",
        "group-hf",
        "group-focus-within",
        "focus-within",
        "hf",
      ],
      textDecoration: [
        "group-hf",
        "hf",
      ],
      textOpacity: [
        "data-current",
        "group-hf",
        "hf",
      ],
    },
  },
  plugins: [
    require("./ui/plugins/interactions"),
    require("./ui/plugins/forms"),
    require("./ui/plugins/prose"),
  ],
}

module.exports = config
