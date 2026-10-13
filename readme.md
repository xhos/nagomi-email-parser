# nagomi-email-parser

An SMTP server that turns forwarded bank notification emails into [nagomi](https://github.com/xhos/nagomi) transactions. Useful for banks with no API. Supports RBC and Wise.

## config

See [.env.example](.env.example). Email bodies are never logged.

## setup

Forward bank emails to `<user id>@<DOMAIN>`. The user ID is on the settings page in nagomi-web.

If the forwarding needs a confirmation link, turn on `UNSAFE_SAVE_EML` for long enough to read it from the saved `.eml`.

`get-certs.sh` gets a Let's Encrypt cert through Cloudflare (`CLOUDFLARE_API_TOKEN`, `LETSENCRYPT_EMAIL`). Neither it nor the service renews certs. Restarting is harmless, since senders retry.

## adding a bank

1. Add a package under `internal/email/`.
2. Implement `parser.Parser` from `internal/parser/types.go`.
3. Call `parser.Register` in its `init()`.
4. Blank-import it in `internal/email/all/all.go`.
5. Add tests with sample emails. `UNSAFE_SAVE_EML` is the easy way to get them.

## dev

```bash
run
tst
fmt
regen
bump-protos
```
