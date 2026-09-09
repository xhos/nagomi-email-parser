# nagomi-email-parser

nagomi-email-parser is a stateless microservice that exposes and SMTP server as a way to automaticly ingest transactions into [nagomi-core](https://github.com/xhos/nagomi-core) via bank emails. The service has an easily extensible parser system which allows adding support for new banks with minimal effort. The inteded way to use this is to setup bank email notifications to be forwarder to your personal email inbox, and then have those emails forwarded to this service. This service will then parse the emails and send the transaction data to nagomi-core via its gRPC API.

## why?

some banks do not have an API or any clean way for accessing transactions. Emails are often the only method to keep the transaction data up to date automaticly. So here we are, parsing emails to get the data we need. Like cavemen.

## config

All configuration is done via environment variables.

| variable                        | description                            | default            | required?  |
|---------------------------------|----------------------------------------|--------------------|------------|
| `API_KEY`                       | authentication key for nagomi-core       |                    | [x]        |
| `NAGOMI_CORE_URL`                 | nagomi-core backend url                  |                    | [x]        |
| `DOMAIN`                        | email domain to serve                  |                    | [x]        |
| `SMTP_PORT`                     | smtp server address                    | `127.0.0.1:2525`   | [ ]        |
| `GRPC_PORT`                     | grpc health check address              | `127.0.0.1:55557`  | [ ]        |
| `TLS_KEY`                       | tls private key file path              |                    | [ ]        |
| `LOG_LEVEL`                     | log level (debug, info, warn, error)   | `info`             | [ ]        |
| `LOG_FORMAT`                    | log format (json, text)                | `text`             | [ ]        |
| `TLS_CERT`                      | tls certificate file path              |                    | [ ]        |
| `UNSAFE_DISABLE_TLS_REQUIRED`   | allow opportunistic TLS                | `false`            | [ ]        |
| `UNSAFE_SAVE_EML`               | save incoming emails as .eml files     | `false`            | [ ]        |

- `SMTP_PORT` and `GRPC_PORT` can be specified as just the port number (e.g., `2525`), with colon prefix (`:2525`), or as full address (`0.0.0.0:2525`)
- by default, services bind to `127.0.0.1` (localhost only) for security. use `0.0.0.0:port` to expose externally
- when `TLS_CERT` and `TLS_KEY` are provided, TLS is required by default. set `UNSAFE_DISABLE_TLS_REQUIRED` to allow opportunistic TLS (accept non-TLS connections)
- when `LOG_FORMAT=json`, logs are written to both stdout and `nagomi-email-parser.log` in the working directory. useful for pointing monitoring tools at the log file
- email body content is never logged for privacy/security reasons. use `UNSAFE_SAVE_EML` to save emails to disk for debugging parsers
- parsing failures are logged at ERROR level for visibility in monitoring

## setup

when setting up your bank to forward emails to this service, use the email address format `uuid@your-domain.com`, where `uuid` is your nagomi-core user ID. This allows the service to associate incoming emails with the correct user account. You can obtain your UUID from nagomi-core logs or the settings page in nagomi-web.

most email providers, when you set up forwarding, require you to confirm it by clicking a link in the email. you can see the confirmation link by setting `UNSAFE_SAVE_EML` to save emails as .eml files, then opening them in a text editor. This is intended for one-time forwarding setup, not constant use.

if your domain is hosted on Cloudflare, you can use the `get-certs.sh` script provided to obtain said TLS certs for your domain by using Cloudflare's API. You will need to set `CLOUDFLARE_API_TOKEN` and `LETSENCRYPT_EMAIL` for the script to work. Point `TLS_CERT` and `TLS_KEY` to the obtained cert files. The script will not handle renewals.

> [!IMPORTANT]
> neither the service itself or get-certs.sh handle automatic renewal of TLS certificates. You will need to use whatever method that makes sense for your enviroment to update the certs file and restart the service. This service handles downtime well, since emails will be re-tried by the sending server.

## development

- `UNSAFE_SAVE_EML` is useful for developing new parsers, it saves incoming emails as `.eml` files in the `emails/` directory for later inspection.
- it is recommended to use the provided nix flake for acess to development scripts.

### adding new parsers

it is quite easy to add new parsers for different banks, as long as you know a bit of go/regex, or willing to spend some time prompting it into existence.

1. create a new package under `internal/email/` (e.g., `internal/email/yourbank`).
2. implement the `parser.Parser` interface from `internal/parser/types.go`.
3. register your new parser in an `init()` function within your new package (e.g., `parser.Register(&yourBankParser{})`).
4. add a blank import for your new parser package in `internal/email/all/all.go`.
5. write tests for your new parser, include test data (email objects can be obtained in debug mode).

the added parser should work without any other changes.

contributions are highly welcome, as it's not feasible for me to cover banks I don't use myself.

## 🌱 ecosystem

- [nagomi-core](https://github.com/xhos/nagomi-core) - main backend service
- [nagomi-web](https://github.com/xhos/nagomi-web) - frontend web application
- [nagomi-mobile](https://github.com/xhos/nagomi-mobile) - mobile appplication
- [nagomi-protos](https://github.com/xhos/nagomi-protos) - shared protobuf definitions
- [nagomi-receipts](https://github.com/xhos/nagomi-receipts) - receipt parsing microservice
- [nagomi-email-parser](https://github.com/xhos/nagomi-email-parser) - email parsing service
