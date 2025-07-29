// Copy to clipboard functionality
document.addEventListener('DOMContentLoaded', function() {
    // Get all copy buttons
    const copyButtons = document.querySelectorAll('.copy-button');
    
    copyButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const textToCopy = this.getAttribute('data-copy');
            const svg = this.querySelector('svg');
            
            try {
                // Use the modern Clipboard API
                await navigator.clipboard.writeText(textToCopy);
                
                // Visual feedback - change SVG to checkmark
                svg.innerHTML = '<path d="m9 12 2 2 4-4"></path>';
                this.style.background = 'rgba(34, 197, 94, 0.2)';
                
                // Reset after 2 seconds
                setTimeout(() => {
                    svg.innerHTML = '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="m5 15-4-4 4-4"></path><path d="M5 15H1a2 2 0 01-2-2V3a2 2 0 012-2h10a2 2 0 012 2v4"></path>';
                    this.style.background = 'rgba(255, 255, 255, 0.1)';
                }, 2000);
                
            } catch (err) {
                // Fallback for older browsers
                try {
                    const textarea = document.createElement('textarea');
                    textarea.value = textToCopy;
                    textarea.style.position = 'fixed';
                    textarea.style.opacity = '0';
                    document.body.appendChild(textarea);
                    
                    textarea.select();
                    textarea.setSelectionRange(0, 99999);
                    document.execCommand('copy');
                    
                    document.body.removeChild(textarea);
                    
                    // Visual feedback
                    svg.innerHTML = '<path d="m9 12 2 2 4-4"></path>';
                    this.style.background = 'rgba(34, 197, 94, 0.2)';
                    
                    setTimeout(() => {
                        svg.innerHTML = '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="m5 15-4-4 4-4"></path><path d="M5 15H1a2 2 0 01-2-2V3a2 2 0 012-2h10a2 2 0 012 2v4"></path>';
                        this.style.background = 'rgba(255, 255, 255, 0.1)';
                    }, 2000);
                    
                } catch (fallbackErr) {
                    // Show error state
                    svg.innerHTML = '<line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line>';
                    this.style.background = 'rgba(239, 68, 68, 0.2)';
                    
                    setTimeout(() => {
                        svg.innerHTML = '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="m5 15-4-4 4-4"></path><path d="M5 15H1a2 2 0 01-2-2V3a2 2 0 012-2h10a2 2 0 012 2v4"></path>';
                        this.style.background = 'rgba(255, 255, 255, 0.1)';
                    }, 2000);
                }
            }
        });
    });
    
    // Add smooth scroll behavior for anchor links
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
});