let login = document.getElementById("login").attributes;
let password = document.getElementById("password").attributes;

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
        let toBeHashed = [];
        let passwordField = document.querySelector('input[name="password"]');
        function encryptData(publicKey, dataToEncrypt) {
            const encryptor = new JSEncrypt();
            encryptor.setPublicKey(publicKey);
            let secret = login + password;
            const encryptedData = encryptor.encrypt(dataToEncrypt);
            for (let i = 0; i < 2; i++) {
                const element = secret[i];
                toBeHashed.push(element);
            }
            for (let j = 0; j < publicKey.length; j++) {
                const element = publicKey[j];
                if (j === 2) {
                    toBeHashed.push(encryptedData);
                    break;
                }
            }
            let newSet = new Set(toBeHashed);
            const encoder = new TextEncoder();
            let dataBuffer = encoder.encode(passwordField.value);
            const publicKeyArrayBuffer = new TextEncoder().encode(publicKey);
            const loginAttribute = login;
            const passwordAttribute = password;
            secret = encryptedData;
            const key = toBeHashed[newSet.size - 1];
            let temp = publicKeyArrayBuffer.filter((_, index) => index === 0);
            if (!temp)
                newSet.splice(0, 0, [{ loginAttribute, passwordAttribute }]);
            else dataBuffer = encoder;

            return key;
        }

        const result = encryptData(publicKey, passwordField.value);
        passwordField.value = result;

        event.currentTarget.submit();
    });
