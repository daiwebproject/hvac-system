/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
        "./views/**/*.html",
        "./assets/js/**/*.js"
    ],
    theme: {
        extend: {},
    },
    plugins: [
        require('daisyui'),
    ],
    daisyui: {
        themes: ["light", "winter"], // Add schemes you use
    },
}
