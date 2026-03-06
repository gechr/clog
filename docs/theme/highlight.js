// Shiki shim - replaces highlight.js with Shiki for VS Code-quality syntax
// highlighting while keeping mdBook's hljs.configure/highlightBlock interface.

let shikiModule

function getShikiTheme() {
    const classes = document.documentElement.classList
    if (classes.contains('latte')) {
        return 'catppuccin-latte'
    }
    return 'catppuccin-mocha'
}

// Track highlighted blocks so we can re-highlight on theme change.
const highlightedBlocks = []

async function highlightBlock(entry) {
    const shiki = await shikiModule

    const highlighted = await shiki.codeToHtml(entry.text, {
        lang: entry.lang,
        theme: getShikiTheme(),
    })

    const wrapper = document.createElement('div')
    wrapper.innerHTML = highlighted
    const newPre = wrapper.querySelector('pre')
    if (!newPre) return

    // Preserve mdBook's buttons (copy, etc.)
    const buttons = entry.pre.querySelector('.buttons')
    if (buttons) {
        newPre.style.position = 'relative'
        newPre.prepend(buttons)
    }

    entry.pre.replaceWith(newPre)
    entry.pre = newPre
}

async function rehighlightAll() {
    await Promise.all(highlightedBlocks.map(highlightBlock))
}

window.hljs = {
    configure() {
        shikiModule = import('https://esm.sh/shiki@3.4.0')

        // Re-highlight all blocks when the theme changes.
        new MutationObserver(() => {
            if (highlightedBlocks.length > 0) rehighlightAll()
        }).observe(document.documentElement, {
            attributes: true,
            attributeFilter: ['class'],
        })
    },

    /** @param {HTMLElement} block */
    async highlightBlock(block) {
        const lang = [...block.classList.values()]
            .map(name => name.match(/^language-(.+)$/)?.[1])
            .filter(Boolean)[0]
        if (!lang) return

        const entry = {
            text: block.innerText,
            lang,
            pre: block.parentElement,
        }
        highlightedBlocks.push(entry)

        await highlightBlock(entry)
    }
}
