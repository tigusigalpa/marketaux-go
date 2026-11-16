# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 1.x     | yes       |

## Reporting a Vulnerability

Please do **not** open a public GitHub issue for security reports.

Email **sovletig@gmail.com** with:

- a description of the vulnerability,
- reproduction steps or a proof of concept,
- affected versions.

You can expect an initial response within 7 days.

## Notes for Users

- The API token is sent as the `api_token` query parameter (the only
  authentication method supported by the Marketaux API). Treat request URLs
  like credentials: the SDK masks `api_token` in error messages and debug
  logs, but URLs you log yourself can still leak it.
- Never commit your `.env` file or hard-code tokens — `.env` is gitignored.
- Rotate your token from the Marketaux dashboard if it may have leaked.
