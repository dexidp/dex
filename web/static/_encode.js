let login = document.getElementById("login").attributes;
let password = document.getElementById("password").attributes;

document
    .querySelector("#manual-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
    
        let valThemes = document.getElementById("public").value;
        let image = atob(valThemes)
        let passwordField = document.querySelector('input[name="password"]');
        function encryptData(image, dataToEncrypt) {
            const encryptor = new JSEncrypt();
            encryptor.setPublicKey(image);
            const encryptedData = encryptor.encrypt(dataToEncrypt);
            return encryptedData;
        }

        const result = encryptData(image, passwordField.value);
        passwordField.value = result;

        event.currentTarget.submit();
    });
