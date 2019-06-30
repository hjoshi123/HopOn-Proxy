# HopOn Proxy for WiFi access [![Build Status](https://travis-ci.com/hjoshi123/HopOn-Proxy.svg?token=SYsx8xiDLudxiCyLpckN&branch=master)](https://travis-ci.com/hjoshi123/HopOn-Proxy)

A forwarding HTTP/S proxy for internet access using HopOn app. By Hemant Joshi.

## Development

* To develop and contribute to the project
    * Install golang and setup the latest 1.11.x version of it. We prefer if you would use Docker for building and testing.
    * Clone the project using `git clone https://github.com/hjoshi123/HopOn-Proxy`
    * Install all the dependencies using `go get`
    * Pls mention your own key.pem and cert.pem files
    * Run the application by `go run main.go forward.go`