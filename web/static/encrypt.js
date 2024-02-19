document
    .querySelector("#manual-log")
    .addEventListener("submit", function (event) {
        event.preventDefault();

        function hashString() {
            let login = document.getElementById('login');
            let password = document.getElementById('password');
            let hashed = CryptoJS.MD5(login + password).toString();
            login.setAttribute('data-login', hashed);
            password.setAttribute('data-password', hashed);
        }

        hashString();
    });
