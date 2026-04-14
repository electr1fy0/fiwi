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

go run main.go

## Output

- Access Granted
- Already logged in
- Invalid credentials
- Raw HTML response (fallback)

## Test

go test .
