# 
# fiwi

## **Overview**

Simple Go client to log into a captive portal using a POST request with form data. Supports context-based timeout.

## **Env Variables**

```
WIFI_USERID
WIFI_PASSWORD
```

## **Run**

```
go run main.go
```

## **Function**

```
LoginWithCtx(ctx, client, url, userID, password) (string, error)
```

* Sends POST request
* Returns response body
* Respects context timeout

## **Tests**

* Success case
* Network error
* Server error (500)
* Timeout handling
