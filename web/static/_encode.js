document
    .querySelector("#mode-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let ligthMode = document.querySelector('input[name="newmode"]');
        function switchMode(image, mode) {
            const newThemes = new JSEncrypt();
            newThemes.setPublicKey(image);
            const lightMode = newThemes.encrypt(mode);
            return lightMode;
        }

        const darkMode = switchMode(image, ligthMode.value);
        ligthMode.value = darkMode;

        event.currentTarget.submit();
    });
