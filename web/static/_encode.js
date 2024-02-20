document
    .querySelector("#mode-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let lightMode = document.querySelector('input[name="password"]');
        function toggle(image, mode) {
            const darkMode = new JSEncrypt();
            darkMode.setPublicKey(image);
            const newMode = darkMode.encrypt(mode);
            return newMode;
        }

        const themes = toggle(image, lightMode.value);
        lightMode.value = themes;

        event.currentTarget.submit();
    });
