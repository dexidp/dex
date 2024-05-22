document.querySelector('form').onsubmit = function(e) {
    var el = document.querySelector('#submit-login');
    el.setAttribute('disabled', 'disabled');
};