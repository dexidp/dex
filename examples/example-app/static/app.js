(function() {
    const crossClientInput = document.getElementById("cross_client_input");
    const crossClientList = document.getElementById("cross-client-list");
    const addClientBtn = document.getElementById("add-cross-client");
    const scopesList = document.getElementById("scopes-list");
    const customScopeInput = document.getElementById("custom_scope_input");
    const addCustomScopeBtn = document.getElementById("add-custom-scope");

    // Default scopes that should be checked by default
    const defaultScopes = ["openid", "profile", "email", "offline_access"];

    // Check default scopes on page load
    document.addEventListener("DOMContentLoaded", function() {
        const checkboxes = scopesList.querySelectorAll('input[type="checkbox"]');
        checkboxes.forEach(cb => {
            if (defaultScopes.includes(cb.value)) {
                cb.checked = true;
            }
        });
    });

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
})();

