(function() {
    const crossClientInput = document.getElementById("cross_client_input");
    const crossClientList = document.getElementById("cross-client-list");
    const addClientBtn = document.getElementById("add-cross-client");
    const scopesList = document.getElementById("scopes-list");
    const customScopeInput = document.getElementById("custom_scope_input");
    const addCustomScopeBtn = document.getElementById("add-custom-scope");

    // Default scopes that should be checked by default
    const defaultScopes = ["openid", "profile", "email", "offline_access"];

    // The script is loaded on every page; only the index has these controls.
    if (scopesList) {
        scopesList.querySelectorAll('input[type="checkbox"]').forEach(cb => {
            if (defaultScopes.includes(cb.value)) {
                cb.checked = true;
            }
        });
    }

    function addCrossClient(value) {
        const trimmed = value.trim();
        if (!trimmed) return;

        const chip = document.createElement("div");
        chip.className = "chip";

        const text = document.createElement("span");
        text.textContent = trimmed;

        const hidden = document.createElement("input");
        hidden.type = "hidden";
        hidden.name = "cross_client";
        hidden.value = trimmed;

        const remove = document.createElement("button");
        remove.type = "button";
        remove.textContent = "×";
        remove.onclick = () => crossClientList.removeChild(chip);

        chip.append(text, hidden, remove);
        crossClientList.appendChild(chip);
    }

    function addCustomScope(scope) {
        const trimmed = scope.trim();
        if (!trimmed || !scopesList) return;

        // Check if scope already exists
        const existingCheckboxes = scopesList.querySelectorAll('input[type="checkbox"]');
        for (const cb of existingCheckboxes) {
            if (cb.value === trimmed) {
                cb.checked = true;
                return;
            }
        }

        // Add new scope checkbox
        const scopeItem = document.createElement("div");
        scopeItem.className = "scope-item";

        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.name = "extra_scopes";
        checkbox.value = trimmed;
        checkbox.id = "scope_custom_" + trimmed;
        checkbox.checked = true;

        const label = document.createElement("label");
        label.htmlFor = checkbox.id;
        label.textContent = trimmed;

        scopeItem.append(checkbox, label);
        scopesList.appendChild(scopeItem);
    }

    addClientBtn?.addEventListener("click", () => {
        addCrossClient(crossClientInput.value);
        crossClientInput.value = "";
        crossClientInput.focus();
    });

    crossClientInput?.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
            e.preventDefault();
            addCrossClient(crossClientInput.value);
            crossClientInput.value = "";
        }
    });

    addCustomScopeBtn?.addEventListener("click", () => {
        addCustomScope(customScopeInput.value);
        customScopeInput.value = "";
        customScopeInput.focus();
    });

    customScopeInput?.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
            e.preventDefault();
            addCustomScope(customScopeInput.value);
            customScopeInput.value = "";
        }
    });

    // Scope pickers on the flow forms take custom scopes too.
    document.querySelectorAll(".add-custom-scope").forEach(function (btn) {
        var wrap = btn.closest(".form-control");
        var input = wrap && wrap.querySelector(".custom-scope-input");
        var list = wrap && wrap.querySelector(".scopes-list");
        if (!input || !list) return;

        var add = function () {
            var value = input.value.trim();
            if (!value) return;

            var existing = list.querySelector('input[value="' + value + '"]');
            if (existing) {
                existing.checked = true;
            } else {
                var item = document.createElement("div");
                item.className = "scope-item";
                var box = document.createElement("input");
                box.type = "checkbox";
                box.name = "scopes";
                box.value = value;
                box.checked = true;
                box.id = "scope_custom_" + value;
                var label = document.createElement("label");
                label.htmlFor = box.id;
                label.textContent = value;
                item.append(box, label);
                list.appendChild(item);
            }
            input.value = "";
            input.focus();
        };

        btn.addEventListener("click", add);
        input.addEventListener("keydown", function (e) {
            if (e.key === "Enter") {
                e.preventDefault();
                add();
            }
        });
    });

    // Device Grant Login Handler
    const deviceGrantBtn = document.getElementById("device-grant-btn");
    deviceGrantBtn?.addEventListener("click", async () => {
        deviceGrantBtn.disabled = true;
        deviceGrantBtn.textContent = "Loading...";

        try {
            // Collect form data similar to regular login
            const form = document.getElementById("login-form");
            const formData = new FormData(form);

            // Get selected scopes
            const scopes = formData.getAll("extra_scopes");

            // Get cross-client values
            const crossClients = formData.getAll("cross_client");

            // Get connector_id if specified
            const connectorId = formData.get("connector_id") || "";

            // Initiate device flow with options
            const response = await fetch('/device/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    scopes: scopes,
                    cross_clients: crossClients,
                    connector_id: connectorId
                })
            });

            if (response.ok) {
                // Redirect to device flow page
                window.location.href = '/device';
            } else {
                const errorText = await response.text();
                alert('Failed to start device flow: ' + errorText);
            }
        } catch (error) {
            alert('Error starting device flow: ' + error.message);
        } finally {
            deviceGrantBtn.disabled = false;
            deviceGrantBtn.textContent = "Device code";
        }
    });
})();


// JSON syntax highlighting. Claims and API responses are read, not skimmed:
// telling a key from a value from a number is most of what makes a blob of
// JSON legible at a glance.
(function () {
    function escapeHTML(text) {
        return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }

    function highlight(text) {
        return escapeHTML(text).replace(
            /("(\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\b\d+(\.\d+)?([eE][+-]?\d+)?\b)/g,
            function (match) {
                var cls = "json-number";
                if (/^"/.test(match)) {
                    cls = /:$/.test(match) ? "json-key" : "json-string";
                } else if (/true|false/.test(match)) {
                    cls = "json-boolean";
                } else if (/null/.test(match)) {
                    cls = "json-null";
                }
                return '<span class="' + cls + '">' + match + "</span>";
            },
        );
    }

    document.querySelectorAll("pre.json").forEach(function (el) {
        var text = el.textContent;
        try {
            // Reject anything that is not JSON rather than colouring it wrongly:
            // these blocks also carry error text from the provider.
            JSON.parse(text);
        } catch (e) {
            return;
        }
        el.innerHTML = highlight(text);
    });

    // A JWT is three base64 segments; colouring the dots apart is enough to see
    // where the header ends and the signature begins.
    document.querySelectorAll("pre.token[data-jwt]").forEach(function (el) {
        var parts = el.textContent.trim().split(".");
        if (parts.length !== 3) return;
        el.innerHTML =
            '<span class="jwt-header">' + escapeHTML(parts[0]) + "</span>." +
            '<span class="jwt-payload">' + escapeHTML(parts[1]) + "</span>." +
            '<span class="jwt-signature">' + escapeHTML(parts[2]) + "</span>";
    });
})();

// "Use my access token" and friends fill the field they sit under, so trying a
// tool against a different one of your tokens is a click rather than a paste.
document.querySelectorAll(".fill-token").forEach(function (btn) {
    btn.addEventListener("click", function () {
        var control = btn.closest(".form-control");
        var field = control && control.querySelector("textarea");
        if (!field) return;
        field.value = btn.getAttribute("data-token");
        field.focus();
    });
});

// List fields — redirect URIs, trusted peers, allowed connectors — are lists in
// the API, so the form collects them as one value each rather than as lines of
// text somebody has to split.
document.querySelectorAll(".chip-add").forEach(function (btn) {
    var control = btn.closest(".form-control");
    var list = control && control.querySelector(".chips");
    var input = control && control.querySelector(".chip-input");
    if (!list || !input) return;

    var add = function () {
        var value = input.value.trim();
        if (!value) return;

        var chip = document.createElement("span");
        chip.className = "chip";

        var text = document.createElement("span");
        text.textContent = value;

        var hidden = document.createElement("input");
        hidden.type = "hidden";
        hidden.name = list.getAttribute("data-name");
        hidden.value = value;

        var remove = document.createElement("button");
        remove.type = "button";
        remove.className = "chip-remove";
        remove.textContent = "×";
        remove.addEventListener("click", function () { chip.remove(); });

        chip.append(text, hidden, remove);
        list.appendChild(chip);
        input.value = "";
        input.focus();
    };

    btn.addEventListener("click", add);
    input.addEventListener("keydown", function (e) {
        if (e.key === "Enter") {
            e.preventDefault();
            add();
        }
    });
});

document.querySelectorAll(".chip-remove").forEach(function (btn) {
    btn.addEventListener("click", function () {
        var chip = btn.closest(".chip");
        if (chip) chip.remove();
    });
});

// Back-channel logout arrives over an event stream rather than on the next page
// load, because when it arrives is the whole point: a push that only surfaces
// when you reload is indistinguishable from the prompt=none check this app
// already runs on every load.
(function () {
    if (!window.EventSource) return;

    // Only where the notice has something to say. A stream is a connection held
    // open for as long as the page lives, and a browser allows six of them per
    // host, so opening one from every page — the admin screens included — is how
    // you starve the rest of the site of connections.
    if (!document.getElementById("signed-in-card")) return;

    const stream = new EventSource("/events");

    // Release the connection as the page goes away rather than waiting for the
    // browser to notice, so a reload does not briefly hold two.
    window.addEventListener("pagehide", function () {
        stream.close();
    });

    stream.addEventListener("backchannel-logout", function (event) {
        let notice = {};
        try {
            notice = JSON.parse(event.data);
        } catch (e) {
            return;
        }

        const container = document.querySelector(".container");
        if (!container || document.getElementById("backchannel-banner")) return;

        // The session is over, so the page must stop showing one. Leaving the
        // signed-in card up next to a notice saying you are signed out is the
        // page contradicting itself.
        document.getElementById("signed-in-card")?.remove();
        document.getElementById("signed-out-card")?.removeAttribute("hidden");

        const card = document.createElement("div");
        card.className = "card";
        card.id = "backchannel-banner";

        const title = document.createElement("div");
        title.className = "card-title";
        title.textContent = "Signed out by back-channel logout";

        const hint = document.createElement("p");
        hint.className = "hint";
        const at = notice.at ? new Date(notice.at).toLocaleTimeString() : "just now";
        hint.textContent =
            "dex pushed a logout token to this app at " + at +
            ". Nothing was loaded here to find that out" +
            (notice.sid ? " — session " + notice.sid : "") + ".";

        const actions = document.createElement("div");
        actions.className = "form-actions";

        const dismiss = document.createElement("button");
        dismiss.type = "button";
        dismiss.className = "button button-secondary";
        dismiss.textContent = "Dismiss";
        dismiss.addEventListener("click", function () { card.remove(); });

        actions.append(dismiss);
        card.append(title, hint, actions);
        container.insertBefore(card, container.querySelector(".card"));
    });
})();
