/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        canvas: 'rgb(var(--canvas) / <alpha-value>)',
        surface: {
          DEFAULT: 'rgb(var(--surface) / <alpha-value>)',
          2: 'rgb(var(--surface-2) / <alpha-value>)',
          3: 'rgb(var(--surface-3) / <alpha-value>)',
          deep: 'rgb(var(--surface-deep) / <alpha-value>)',
          hover: 'rgb(var(--surface-hover) / <alpha-value>)',
          active: 'rgb(var(--surface-active) / <alpha-value>)'
        },
        fg: {
          DEFAULT: 'rgb(var(--fg) / <alpha-value>)',
          muted: 'rgb(var(--fg-muted) / <alpha-value>)',
          subtle: 'rgb(var(--fg-subtle) / <alpha-value>)'
        },
        line: {
          DEFAULT: 'rgb(var(--line) / var(--line-a))',
          strong: 'rgb(var(--line) / var(--line-a-strong))'
        },
        glass: {
          DEFAULT: 'rgb(var(--glass) / var(--glass-a))',
          strong: 'rgb(var(--glass) / var(--glass-a-strong))'
        },
        spinner: 'rgb(var(--spinner) / <alpha-value>)'
      },
      boxShadow: {
        'glow-primary': '0 0 20px -5px rgba(99, 102, 241, 0.4)',
      },
    },
  },
  plugins: [],
}
