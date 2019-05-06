Please tag me @adai to enable finding replies to the questions easily.

The following questions are all pertaining to the main.go file in sfu-minimal code example at pion/webrtc. Refer here: https://github.com/pion/webrtc/blob/master/examples/sfu-minimal/main.go 

1. Will the `peerConnection.OnTrack()` function on line 58 be triggered after we have copied the `answer` from line 110 (`fmt.Println(signal.Encode(answer))`) into the jsfiddle.net browser and clicked start session in the browser? In other words precisely when will the `OnTrack` function be called?

2. Will the `peerConection.OnTrack()` function be called whenever 
- a video uploader joins the server?
- a video downloading browser joins the server?

3. To whom does the `receiver *webrtc.RTPReceiver` function argument refer to in line 58. Does it refer to the server or the remote browser?

4. Why are we using a channel, namely `localTrackChan`, instead of a normal variable, to transfer the `localTrack` info from line 75 to line 112 of main.go? Is this to block the program on line 112, main.go, to ensure that we have a broadcaster (i.e., a browser which is uploading video to the server) before there is any client to download the video from the server?

5. Will the program still work fine if browser downloaders join the server first before any video uploaders are available?

6. Upon executing the following statement: `peerConnection.SetRemoteDescription(offer)`, line 92, is there any network communication which occurs(e.g., communicate to remote browser or communicate to ICE/STUN)? What type of network communication occurs and what is their purpose?

7. At lines 45 and 121, new `peerConnection`s are formed for each uploader and downloader browser. Hence, we are overwriting the previous `peerConnection` variable each time we declare a new one. By overwriting the previous `peerConnection` handles, we will be unable to change any properties of those peer-to-peer connections since we have lost their handles, right? 


The following are questions from other webrtc library.

8.  Refer here: https://github.com/pion/webrtc/blob/3d2c1c2c32c96b5124c43ad85390cf1fb0961924/track.go#L81 . In `func (t *Track)Read(b []byte) (n int, err error)`, why when number of `len(t.activeSenders) != 0`, it implies a local track?

9. Within the `func (pc *PeerConnection) SetRemoteDescription(desc SessionDescription) error` function, what is the purpose and function of the goroutine at this line: https://github.com/pion/webrtc/blob/3d2c1c2c32c96b5124c43ad85390cf1fb0961924/peerconnection.go#L984 

Thank you for your help.