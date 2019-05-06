# Introduction

This is an extension of the SFU-Minimal code example from [github.com/pion/webrtc](https://github.com/pion/webrtc/tree/master/examples/sfu-minimal).

In the original SFU-Minimal code example, the `Publish` and `Join` webpages are hosted on jsfiddle.net. We need to manually transfer the SDPs between the webpage and our server.

In this extension, we strive to build a signalling server and automate the exchange of SDPs between the webpages and the server. Moreover, the webpages `Publish` and `Join` are hosted on the same server as the signalling server.

Additionally, we try to remember the publisher and client so as to reconnect upon a loss of connection.

# Usage

1. 
Ensure `GO111MODULE=on` in your terminal.
1. To run locally:
    + Run `go install`.
    + Then run the executable, i.e., `webrtc`.
1. To run the code in Docker, do the following:
    + Run `docker build -t webrtc .` in the webrtc folder.
    + Then run `docker-compose up` to     
1. Go to `localhost:8088/publish` web page which will start capturing video using your webcam. This video will be broadcast to multiple clients.
1. Then open another tab in your browser, and go to `localhost:8088/join` to see the broadcasted video. 
