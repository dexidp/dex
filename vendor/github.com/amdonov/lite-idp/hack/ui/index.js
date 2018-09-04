require("./node_modules/patternfly/dist/css/patternfly.css");
require("./node_modules/patternfly/dist/css/patternfly-additions.css");
require("./style.css");
require("bootstrap");
// Read a page's GET URL variables and return them as an associative array.
function getUrlVars() {
    var vars = [],
        hash;
    var hashes = window.location.href.slice(window.location.href.indexOf('?') + 1).split('&');
    for (var i = 0; i < hashes.length; i++) {
        hash = hashes[i].split('=');
        vars.push(hash[0]);
        vars[hash[0]] = hash[1];
    }
    return vars;
}
var params = getUrlVars();
$('#requestId').val(params['requestId']);
function urldecode(str) {
    return decodeURIComponent((str+'').replace(/\+/g, '%20'));
}
if ('error' in params) {
    $("#errorBox").removeClass('hidden');
    $("#errorMsg").html(urldecode(params['error']));
}