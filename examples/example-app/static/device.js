(function() {
    const sessionID = document.getElementById("session-id")?.value;
    const deviceCode = document.getElementById("device-code")?.value;
    const pollInterval = parseInt(document.getElementById("poll-interval")?.value || "5", 10);
    const verificationURL = document.getElementById("verification-url")?.textContent;
    const userCode = document.getElementById("user-code")?.textContent;
    const statusText = document.getElementById("status-text");
    const errorMessage = document.getElementById("error-message");
    const openAuthBtn = document.getElementById("open-auth-btn");

    let pollTimer = null;

    document.querySelectorAll(".copy-btn").forEach(btn => {
        btn.addEventListener("click", async function() {
            const targetId = this.getAttribute("data-copy");
            const targetElement = document.getElementById(targetId);

            if (targetElement) {
                const textToCopy = targetElement.textContent;

                try {
                    await navigator.clipboard.writeText(textToCopy);
                    const originalText = this.textContent;
                    this.textContent = "✓";
                    setTimeout(() => {
                        this.textContent = originalText;
                    }, 2000);
                } catch (err) {
                    console.error('Failed to copy:', err);
                }
            }
        });
    });

    openAuthBtn?.addEventListener("click", () => {
        if (verificationURL && userCode) {
            const url = verificationURL + "?user_code=" + encodeURIComponent(userCode);
            window.open(url, "_blank", "width=600,height=800");
        }
    });

    async function pollForToken() {
        try {
            const response = await fetch('/device/poll', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: sessionID,
                    device_code: deviceCode
                })
            });

            const data = await response.json();

            if (response.ok && data.status === 'complete') {
                statusText.textContent = "Authentication successful! Redirecting...";
                stopPolling();
                window.location.href = '/device/result';
            } else if (response.ok && data.status === 'pending') {
                statusText.textContent = "Waiting for authentication...";
            } else {
                const errorText = data.error_description || data.error || 'Unknown error';

                if (data.error === 'session_expired') {
                    showError('This session has been superseded by a new device flow. Please start over.');
                    stopPolling();
                } else if (data.error === 'expired_token' || data.error === 'access_denied') {
                    showError(data.error === 'expired_token' ?
                        'The device code has expired. Please start over.' :
                        'Authentication was denied.');
                    stopPolling();
                }
            }
        } catch (error) {
            console.error('Polling error:', error);
        }
    }

    function showError(message) {
        errorMessage.textContent = message;
        errorMessage.style.display = 'block';

        // Hide the status indicator (contains spinner and status text)
        const statusIndicator = document.querySelector('.status-indicator');
        if (statusIndicator) {
            statusIndicator.style.display = 'none';
        }
    }

    function startPolling() {
        pollForToken();
        pollTimer = setInterval(pollForToken, pollInterval * 1000);
    }

    function stopPolling() {
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = null;
        }
    }

    if (deviceCode) {
        startPolling();
    }

    window.addEventListener('beforeunload', stopPolling);
})();

