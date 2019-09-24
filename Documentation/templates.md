# Templates

## Using your own templates

Dex supports using your own templates and passing arbitrary data to them to help customize your installation.

Steps:

1. Copy contents of the `web` directory over to a new directory.
2. Customize the templates as needed, be sure to retain all the existing variables so Dex continues working correctly.
  a. Use this syntax `{{ "your_key" | extra }}` to use values from `frontend.extra`.
3. Write a theme for your templates in the `themes` directory.
4. Add your custom data to the Dex configuration `frontend.extra`.
   ```yaml
   frontend:
     dir: /path/to/custom/web
     issuer: my-dex
     extra:
       tos_footer_link: "https://example.com/terms"
       client_logo_url: "../theme/client-logo.png"
       foo: "bar"
   ```
5. Set the `frontend.dir` value to your own `web` directory.
6. Write the issuer in the `issuer` directory in order to modify the Dex title and the `Log in to <<dex>>` tag.

To test your templates simply run Dex with a valid configuration and go through a login flow.