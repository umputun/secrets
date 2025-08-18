// theme switching functionality
(function() {
    'use strict';

    const THEME_KEY = 'safesecret-theme';
    const THEMES = {
        LIGHT: 'light',
        DARK: 'dark',
        AUTO: 'auto'
    };

    const themeToggle = document.getElementById('theme-toggle');
    const themeIcon = document.getElementById('theme-icon');

    if (!themeToggle || !themeIcon) {
        return; // exit if elements not found
    }

    // get current theme from localStorage or default to auto
    function getCurrentTheme() {
        return localStorage.getItem(THEME_KEY) || THEMES.AUTO;
    }

    // get the effective theme (resolves auto to light/dark)
    function getEffectiveTheme(theme = getCurrentTheme()) {
        if (theme === THEMES.AUTO) {
            return window.matchMedia('(prefers-color-scheme: dark)').matches ? THEMES.DARK : THEMES.LIGHT;
        }
        return theme;
    }

    // apply theme to the document
    function applyTheme(theme) {
        const effectiveTheme = getEffectiveTheme(theme);
        
        // remove existing theme attributes
        document.documentElement.removeAttribute('data-theme');
        
        // set new theme if not auto (let CSS handle auto via prefers-color-scheme)
        if (theme !== THEMES.AUTO) {
            document.documentElement.setAttribute('data-theme', effectiveTheme);
        }
        
        // update icon
        updateThemeIcon(theme);
        
        // store preference
        localStorage.setItem(THEME_KEY, theme);
    }

    // update the theme toggle icon
    function updateThemeIcon(theme) {
        const effectiveTheme = getEffectiveTheme(theme);
        
        if (theme === THEMES.AUTO) {
            themeIcon.textContent = '🌓'; // auto mode
        } else if (effectiveTheme === THEMES.DARK) {
            themeIcon.textContent = '☀️'; // show sun when in dark mode (click to go light)
        } else {
            themeIcon.textContent = '🌙'; // show moon when in light mode (click to go dark)
        }
    }

    // cycle through themes: light -> dark -> auto -> light
    function toggleTheme() {
        const currentTheme = getCurrentTheme();
        let nextTheme;
        
        switch (currentTheme) {
            case THEMES.LIGHT:
                nextTheme = THEMES.DARK;
                break;
            case THEMES.DARK:
                nextTheme = THEMES.AUTO;
                break;
            case THEMES.AUTO:
            default:
                nextTheme = THEMES.LIGHT;
                break;
        }
        
        applyTheme(nextTheme);
    }

    // listen for system theme changes when in auto mode
    function handleSystemThemeChange() {
        if (getCurrentTheme() === THEMES.AUTO) {
            applyTheme(THEMES.AUTO);
        }
    }

    // initialize theme on page load
    function initTheme() {
        applyTheme(getCurrentTheme());
    }

    // set up event listeners
    themeToggle.addEventListener('click', toggleTheme);
    
    // listen for system theme changes
    if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', handleSystemThemeChange);
    }

    // initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTheme);
    } else {
        initTheme();
    }
})();