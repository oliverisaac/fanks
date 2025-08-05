const colors = require('tailwindcss/colors')

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    'views/*.templ',
  ],
  theme: {
    container: {
      center: true,
      padding: {
        DEFAULT: '0.1rem',
        sm: '0.3rem',
        lg: '2rem',
        xl: '5rem',
      },
    },
    extend: {
      colors: {
        primary: colors.green,
        secondary: colors.teal,
        neutral: colors.slate,
      }
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ]
}