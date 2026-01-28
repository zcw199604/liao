/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'surface-light': 'rgba(255, 255, 255, 0.08)',
        'surface-lighter': 'rgba(255, 255, 255, 0.12)',
        'border-subtle': 'rgba(255, 255, 255, 0.08)',
      },
      boxShadow: {
        'glow-primary': '0 0 20px -5px rgba(99, 102, 241, 0.4)',
      },
    },
  },
  plugins: [],
}
