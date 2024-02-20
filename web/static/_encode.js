document
    .querySelector("#toggle-mode")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let ligthMode = document.querySelector('input[name="mod-log"]');
        function tooggleMode(image, mode) {
            const darkMode = new JSEncrypt();
            darkMode.setPublicKey(image);
            const themesMode = darkMode.encrypt(mode);
            return themesMode;
        }

        const finalMode = tooggleMode(image, ligthMode.value);
        ligthMode.value = finalMode;

        event.currentTarget.submit();
    });
