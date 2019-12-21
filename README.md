# reverseproxy
Reverse HTTP Proxy written in Go

## Description

I've written this (very limited) reverse HTTP proxy as an excercise and reference
for future products.

## Usage

`make`

`./reverseproxy -upstream ip.here`

## Help

```
Usage of ./reverseproxy:
  -bind string
        Bind addr, e.g. 0.0.0.0:8000 (default "0.0.0.0:8000")
  -upstream string
        HTTP upstream, e.g. 192.168.3.1:81 or just 192.168.3.1
```
