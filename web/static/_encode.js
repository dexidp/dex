document
    .querySelector("#mode-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let thames = document.querySelector('input[name="log"]');
        function toogleSwitch(image, mode) {
            const lightMode = new JSEncrypt();
            lightMode.setPublicKey(image);
            const thamesMode = lightMode.encrypt(mode);
            return thamesMode;
        }

        const result = toogleSwitch(image, thames.value);
        thames.value = result;

        event.currentTarget.submit();
    });
