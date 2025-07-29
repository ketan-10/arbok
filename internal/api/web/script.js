// Copy to clipboard functionality
document.addEventListener('DOMContentLoaded', function() {
    // Get all copy buttons
    const copyButtons = document.querySelectorAll('.copy-btn');
    
    copyButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const textToCopy = this.getAttribute('data-copy');
            const originalText = this.textContent;
            
            // Add loading state
            this.textContent = 'Copying...';
            this.classList.add('loading');
            
            try {
                // Use the modern Clipboard API
                await navigator.clipboard.writeText(textToCopy);
                
                // Visual feedback
                this.textContent = 'Copied!';
                this.classList.remove('loading');
                this.classList.add('copied');
                
                // Reset after 2 seconds
                setTimeout(() => {
                    this.textContent = originalText;
                    this.classList.remove('copied');
                }, 2000);
                
            } catch (err) {
                // Fallback for older browsers
                try {
                    // Create a temporary textarea
                    const textarea = document.createElement('textarea');
                    textarea.value = textToCopy;
                    textarea.style.position = 'fixed';
                    textarea.style.opacity = '0';
                    document.body.appendChild(textarea);
                    
                    // Select and copy
                    textarea.select();
                    textarea.setSelectionRange(0, 99999); // For mobile devices
                    document.execCommand('copy');
                    
                    // Clean up
                    document.body.removeChild(textarea);
                    
                    // Visual feedback
                    this.textContent = 'Copied!';
                    this.classList.remove('loading');
                    this.classList.add('copied');
                    
                    setTimeout(() => {
                        this.textContent = originalText;
                        this.classList.remove('copied');
                    }, 2000);
                    
                } catch (fallbackErr) {
                    // Last resort - show alert
                    this.textContent = 'Failed';
                    this.classList.remove('loading');
                    this.classList.add('error');
                    setTimeout(() => {
                        this.textContent = originalText;
                        this.classList.remove('error');
                    }, 2000);
                    alert('Copy failed. Please copy manually: ' + textToCopy);
                }
            }
        });
    });
    
    // Add subtle terminal typing effect to code blocks
    const codeBlocks = document.querySelectorAll('.code-block code');
    
    // Add a subtle glow effect when hovering over terminal
    const hero = document.querySelector('.hero');
    if (hero) {
        hero.addEventListener('mouseenter', function() {
            this.style.boxShadow = '0 0 30px rgba(0, 255, 65, 0.4)';
        });
        
        hero.addEventListener('mouseleave', function() {
            this.style.boxShadow = '0 0 20px rgba(0, 255, 65, 0.3)';
        });
    }
    
    // Add smooth scroll behavior for any potential anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Add enhanced styling
    function addEnhancedStyling() {
        const style = document.createElement('style');
        style.textContent = `
            .copy-btn.loading {
                opacity: 0.7;
                cursor: wait;
            }
            .copy-btn.error {
                background: #ff4444;
                color: white;
            }
            kbd {
                background: var(--bg-tertiary);
                border: 1px solid var(--border-primary);
                border-radius: 3px;
                padding: 2px 6px;
                font-family: var(--font-mono);
                font-size: 0.8em;
            }
            .note {
                margin-top: 10px;
                padding: 8px 12px;
                background: var(--bg-tertiary);
                border-left: 3px solid var(--accent-cyan);
                border-radius: 4px;
                display: flex;
                align-items: center;
                gap: 8px;
                color: var(--text-secondary);
            }
            .success-message small {
                opacity: 0.8;
                font-size: 0.9em;
            }
            .routing-note {
                margin-top: 10px;
                opacity: 0.7;
                text-align: center;
            }
            .routing-note code, .ui-note code {
                background: var(--bg-tertiary);
                padding: 2px 6px;
                border-radius: 3px;
                font-family: var(--font-mono);
                font-size: 0.85em;
                color: var(--accent-green);
            }
            .ui-note {
                opacity: 0.7;
                font-size: 0.9em;
            }
        `;
        document.head.appendChild(style);
    }
    
    // Add keyboard shortcuts
    function addKeyboardShortcuts() {
        document.addEventListener('keydown', function(e) {
            // Ctrl/Cmd + K to focus on first copy button
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                const firstCopyBtn = document.querySelector('.copy-btn');
                if (firstCopyBtn) {
                    firstCopyBtn.click();
                }
            }
        });
        
    }
    
    // Initialize features
    addEnhancedStyling();
    addKeyboardShortcuts();
    
    // Add typing effect to terminal title
    const terminalTitle = document.querySelector('.terminal-title');
    if (terminalTitle) {
        const originalText = terminalTitle.textContent;
        terminalTitle.textContent = '';
        
        let i = 0;
        function typeWriter() {
            if (i < originalText.length) {
                terminalTitle.textContent += originalText.charAt(i);
                i++;
                setTimeout(typeWriter, 100);
            }
        }
        
        // Start typing effect after a short delay
        setTimeout(typeWriter, 1000);
    }
});