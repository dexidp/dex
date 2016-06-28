# Configuring Sending Emails

Dex sends emails to a during the registration process to verify an email
address belongs to the person signing up. Currently Dex supports two ways of
sending emails, and has a third option for use during development.

Configuration of the email provider in Dex is provided through a JSON file. All
email providers have a `type` and `id` field as well as some additional provider
specific fields.

## SMTP

If using SMTP the `type` field **must** be set to `smtp`. Additionally both
`host` and `port` are required. If you wish to use SMTP plain auth, then
set `auth` to `plain` and specify your username and password.

```
{
    "type": "smtp",
    "host": "smtp.example.org",
    "port": 587,
    "auth": "plain",
    "from": "postmaster@example.com",
    "username": "postmaster@example.org",
    "password": "foo"
}
```

## Mailgun

If using Mailgun the `type` field **must** be set to `mailgun`. Additionally
`privateAPIKey`, `publicAPIKey`, and `domain` are required.

```
{
    "type": "mailgun",
    "from": "noreply@example.com",
    "privateAPIKey": "key-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
    "publicAPIKey": "YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY",
    "domain": "sandboxZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ.mailgun.org"
}
```

## Dev

The fake emailer should only be used in development. The fake emailer
prints emails to `stdout` rather than sending any email. If using the fake
emailer the `type` field **must** be set to `fake`.

```
{
    "type": "fake"
}
```
