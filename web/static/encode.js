document.querySelector('#manual-log').addEventListener('submit', function(event) {
    event.preventDefault();

    var passwordField = document.querySelector('input[name="password"]');
    passwordField.value = btoa(passwordField.value);

    event.currentTarget.submit();
});