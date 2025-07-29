// Copy to clipboard functionality
document.addEventListener('DOMContentLoaded', function() {
    // Get all copy buttons
    const copyButtons = document.querySelectorAll('.copy-btn');
    
    copyButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const textToCopy = this.getAttribute('data-copy');
            const originalText = this.textContent;
            
            try {
                // Use the modern Clipboard API
                await navigator.clipboard.writeText(textToCopy);
                
                // Visual feedback
                this.textContent = 'Copied!';
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
                    this.classList.add('copied');
                    
                    setTimeout(() => {
                        this.textContent = originalText;
                        this.classList.remove('copied');
                    }, 2000);
                    
                } catch (fallbackErr) {
                    // Last resort - show alert
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
    
    // Add matrix-style digital rain effect to background (subtle)
    function createMatrixRain() {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        
        canvas.style.position = 'fixed';
        canvas.style.top = '0';
        canvas.style.left = '0';
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.pointerEvents = 'none';
        canvas.style.zIndex = '-1';
        canvas.style.opacity = '0.05';
        
        document.body.appendChild(canvas);
        
        const resizeCanvas = () => {
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;
        };
        
        resizeCanvas();
        window.addEventListener('resize', resizeCanvas);
        
        const columns = Math.floor(canvas.width / 20);
        const drops = new Array(columns).fill(1);
        
        const chars = '01アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワン';
        
        function drawMatrix() {
            ctx.fillStyle = 'rgba(13, 17, 23, 0.05)';
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            
            ctx.fillStyle = '#00ff41';
            ctx.font = '12px monospace';
            
            for (let i = 0; i < drops.length; i++) {
                const text = chars[Math.floor(Math.random() * chars.length)];
                ctx.fillText(text, i * 20, drops[i] * 20);
                
                if (drops[i] * 20 > canvas.height && Math.random() > 0.975) {
                    drops[i] = 0;
                }
                drops[i]++;
            }
        }
        
        setInterval(drawMatrix, 100);
    }
    
    // Only add matrix effect on larger screens to avoid performance issues
    if (window.innerWidth > 768) {
        createMatrixRain();
    }
    
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