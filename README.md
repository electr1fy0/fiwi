# fiwi

Tiny Go client to log into a captive portal with retries and timeout.

## Credentials

Set env vars:
```bash
WIFI_USERID
WIFI_PASSWORD
```
Or create `~/.fiwi`:

```js
{
“userID”: “your_id”,
“password”: “your_password”
}
```
If neither exists, you’ll be prompted once and it will be saved automatically.

## Run

go run .

Login sends the three form fields `userId`, `password`, and
`serviceName=ProntoAuthentication` to
`http://phc.prontonetworks.com/cgi-bin/authlogin?URI=http://example.com`,
with the browser headers used by the portal's working login request.
The authentication POST does not follow redirects. After login, fiwi checks
`http://connectivitycheck.gstatic.com/generate_204` for HTTP 204 before reporting
internet access.

## Output

- Access Granted
- Already logged in
- Invalid credentials
- Raw HTML response (fallback)

## Test

go test .
