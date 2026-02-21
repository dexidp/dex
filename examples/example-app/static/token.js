// Simple JSON syntax highlighter
document.addEventListener("DOMContentLoaded", function() {
    const claimsElement = document.getElementById("claims");
    if (claimsElement) {
        try {
            const json = JSON.parse(claimsElement.textContent);
            claimsElement.innerHTML = syntaxHighlight(json);
        } catch (e) {
            console.error("Invalid JSON in claims:", e);
        }
    }
});

function syntaxHighlight(json) {
    if (typeof json != 'string') {
        json = JSON.stringify(json, undefined, 2);
    }
    json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return json.replace(/("(\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|\b\d+\b)/g, function (match) {
        let cls = 'number';
        if (/^"/.test(match)) {
            if (/:$/.test(match)) {
                cls = 'key';
            } else {
                cls = 'string';
            }
        } else if (/true|false/.test(match)) {
            cls = 'boolean';
        } else if (/null/.test(match)) {
            cls = 'null';
        }
        return '<span class="' + cls + '">' + match + '</span>';
    });
}

function copyPublicKey() {
    const publicKeyElement = document.getElementById("public-key");
    if (!publicKeyElement) return;

    const text = publicKeyElement.textContent;

    // Use modern clipboard API if available
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
            showCopyFeedback("Copied!");
        }).catch(err => {
            console.error("Failed to copy:", err);
            fallbackCopy(text);
        });
    } else {
        fallbackCopy(text);
    }
}

function fallbackCopy(text) {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    try {
        document.execCommand("copy");
        showCopyFeedback("Copied!");
    } catch (err) {
        console.error("Fallback copy failed:", err);
        showCopyFeedback("Failed to copy");
    }
    document.body.removeChild(textarea);
}

function showCopyFeedback(message) {
    const btn = event.target;
    const originalText = btn.textContent;
    btn.textContent = message;
    btn.style.backgroundColor = "#28a745";
    setTimeout(() => {
        btn.textContent = originalText;
        btn.style.backgroundColor = "";
    }, 2000);
}

// UserInfo functionality
document.addEventListener("DOMContentLoaded", function() {
    const form = document.getElementById("userinfo-form");
    if (form) {
        form.addEventListener("submit", fetchUserInfo);
    }
});

async function fetchUserInfo(event) {
    event.preventDefault();

    const form = event.target;
    const loading = document.getElementById("userinfo-loading");
    const error = document.getElementById("userinfo-error");
    const claimsElement = document.getElementById("userinfo-claims");
    const submitButton = form.querySelector('button[type="submit"]');

    // Hide error and claims from previous attempts
    error.style.display = "none";
    claimsElement.style.display = "none";

    // Show loading, hide button
    submitButton.style.display = "none";
    loading.style.display = "flex";

    try {
        const formData = new FormData(form);

        // Convert FormData to URL-encoded string
        const urlEncodedData = new URLSearchParams(formData).toString();

        const response = await fetch("/userinfo", {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            body: urlEncodedData
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }

        const userinfo = await response.json();

        // Display the userinfo claims
        const code = claimsElement.querySelector("code");
        const formattedJson = JSON.stringify(userinfo, null, 2);
        code.textContent = formattedJson;

        // Apply syntax highlighting
        try {
            code.innerHTML = syntaxHighlight(userinfo);
        } catch (e) {
            console.error("Failed to highlight JSON:", e);
        }

        claimsElement.style.display = "block";

    } catch (err) {
        console.error("Failed to fetch userinfo:", err);
        error.textContent = "Failed to fetch UserInfo: " + err.message;
        error.style.display = "block";
        submitButton.style.display = "inline-block";
    } finally {
        loading.style.display = "none";
    }
}

