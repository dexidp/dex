// Package consent owns the approval (consent) step of the authorization flow:
// the /approval endpoint, the consent screen, recording the user's consent, and
// the decision of whether consent can be skipped.
//
// It is one of the shared flow steps (alongside mfa and issue): the browser
// login flow and, conceptually, any other front-channel flow reach it once the
// user is authenticated. When consent is granted (or already covered) it hands
// off to the issue component to complete the authorization response.
package consent
