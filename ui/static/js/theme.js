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
        
        // Set the SVG attributes to maintain consistent sizing
        themeIcon.setAttribute('width', '28');
        themeIcon.setAttribute('height', '28');
        themeIcon.setAttribute('viewBox', '0 0 24 24');
        
        if (theme === THEMES.AUTO) {
            // auto mode - simplified monitor icon
            themeIcon.innerHTML = '<rect x="3" y="4" width="18" height="12" rx="2" stroke="currentColor" stroke-width="2" fill="none"/><path d="M8 21h8M12 17v4" stroke="currentColor" stroke-width="2"/>';
            themeIcon.style.transform = 'translateX(0px)';
        } else if (effectiveTheme === THEMES.DARK) {
            // show sun when in dark mode (click to go light)
            themeIcon.innerHTML = '<circle cx="12" cy="12" r="4" fill="currentColor"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M17.66 6.34l1.41-1.41" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>';
            themeIcon.style.transform = 'translateX(0px)';
        } else {
            // show moon when in light mode (click to go dark)
            themeIcon.innerHTML = '<path d="M21 12.79C20.25 13.5 19.2 14 18 14C15.79 14 14 12.21 14 10C14 8.8 14.5 7.75 15.21 7C15.14 7 15.07 7 15 7C11.69 7 9 9.69 9 13S11.69 19 15 19S21 16.31 21 13C21 12.93 21 12.86 21 12.79Z" fill="currentColor"/>';
            themeIcon.style.transform = 'translateX(-2px)';
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