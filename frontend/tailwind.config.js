/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#10141c",
        mist: "#edf3f2",
        coral: "#f06f4f",
        cyan: "#6fd8d2",
        sand: "#f5d9aa",
        leaf: "#295348",
      },
      boxShadow: {
        panel: "0 22px 70px rgba(16, 20, 28, 0.16)",
      },
      fontFamily: {
        display: ['"Avenir Next"', '"Segoe UI"', "sans-serif"],
        body: ['"IBM Plex Sans"', '"Trebuchet MS"', "sans-serif"],
        mono: ['"SFMono-Regular"', '"Menlo"', "monospace"],
      },
      backgroundImage: {
        haze:
          "radial-gradient(circle at top left, rgba(111, 216, 210, 0.35), transparent 28%), radial-gradient(circle at top right, rgba(240, 111, 79, 0.24), transparent 22%), linear-gradient(135deg, #f4efe5 0%, #edf3f2 44%, #f8f3ea 100%)",
      },
    },
  },
  plugins: [],
};
