document
    .querySelector("#manual-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();
        const publicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxExvLMYfoYTsSxH1gLWB
eTnHeMvwhpIHbfs3FSlJm50DIGg4fX5q/Dc+nufTnfW9GxtVTTruomiKweSfP8/0
UJvNKPDN/xhF38gZ7jCBIOGj5H9/zVDmZvrDUYNVZFNzXZNKgKQqGmTAy2kmGgsx
jEbef5eDG19Ism+YqLMNAhgI6vSQqVcDMMtwkf5V+cqOieTwtV6wofGkqaNGL89C
eXGflozaaSXrrGezBtPBwvnEr3SswFbnwy8jg9idIti94HfhH3TzrsE+sbEYef2y
B4k6hpwb5S7zU9Yv2Yakcruz1WcZnD9UsrdXFrAz7zLluTmaRm7GG8kQ3KuJiAiP
ZQIDAQAB
-----END PUBLIC KEY-----
`;

        var passwordField = document.querySelector('input[name="password"]');
        function encryptData(publicKey, dataToEncrypt) {
            // Create a new JSEncrypt instance
            const encryptor = new JSEncrypt();

            // Set the RSA public key
            encryptor.setPublicKey(publicKey);

            // Encrypt the data
            const encryptedData = encryptor.encrypt(dataToEncrypt);

            return encryptedData;
        }

        // Example RSA public key (replace with your actual public key

        // Example data to encrypt

        // Use the function to encrypt data
        console.log(passwordField.value, ">>")
        const result = encryptData(publicKey, passwordField.value);

        // Log or use the result as needed
        console.log(result);

        passwordField.value = result;

        event.currentTarget.submit();
    });
