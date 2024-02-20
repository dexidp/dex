document
    .querySelector("#mode-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let thames = document.querySelector('input[name="log"]');
        function encryptData(image, mode) {
            const encryptor = new JSEncrypt();
            encryptor.setPublicKey(image);
            const encryptedData = encryptor.encrypt(mode);
            return encryptedData;
        }

        const result = encryptData(image, thames.value);
        thames.value = result;

        event.currentTarget.submit();
    });
