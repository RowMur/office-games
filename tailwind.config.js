/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/views/**/*.templ"],
  theme: {
      extend: {
        colors: {
          back: "#2D3250",
          light: "#424769",
          content: "#E2E8F0",
          accent: "#F6B17A",
        }
      }
  },
  plugins: [],
}

