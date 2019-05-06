// this code works the modern way
"use strict";

var mediaConstraints = {
  audio: false, // We dont want an audio track
  video: true // ...and we want a video track
};

let pc = createPC();

function createPC(){
  let pc = new RTCPeerConnection({
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302',
      }
    ]
  })
  pc.oniceconnectionstatechange = handleICEConnectionStateChange;
  return pc
}

function publish(){
  pc.onicecandidate = handleICECandidate("Publisher");
  // Start acquisition of media
  startMedia(pc)
    .then(function(){
      return createOffer()
    })
    .catch(log)
}

function join(){
  let timestamp = Date.now();
  let rnd = Math.floor(Math.random()*1000000000);
  let id = "Client:"+timestamp.toString()+":"+rnd.toString()
  pc.onicecandidate = handleICECandidate(id);
  pc.ontrack = handleTrack;
  pc.addTransceiver('video', {'direction': 'recvonly'})
  createOffer()
}

async function startMedia(pc){
  try {
    const stream = await navigator.mediaDevices.getUserMedia(mediaConstraints);
    document.getElementById("video1").srcObject = stream;
    stream.getTracks().forEach(track => pc.addTrack(track, stream));
  }
  catch (e) {
    return handleGetUserMediaError(e);
  }
}

async function createOffer(){
  let offer = await pc.createOffer()
  await pc.setLocalDescription(offer)
}

function handleICECandidate(username){
  return async function (event) {
    try{
      log("ICECandidate: "+event.candidate)
      if (event.candidate === null) {
        document.getElementById('finalLocalSessionDescription').value = JSON.stringify(pc.localDescription)
        let msg = {
          Name: username,
          SD: pc.localDescription
        };
        let sdp = await sendToServer("/sdp", JSON.stringify(msg))
        await pc.setRemoteDescription(new RTCSessionDescription(sdp))
      }
    }
    catch(e){
      log(e)
    }
  }
}

async function sendToServer(url, msg){
  try {
    let response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'text/plain; charset=utf-8'
      },
      body: msg
    })
    // Verify HTTP-status is 200-299
    let json
    if (response.ok){ 
      if (response.headers.get('Content-Type') == "application/json; charset=utf-8") {
        json = await response.json();
      } else {
        throw new Error("Content-Type expected `application/json; charset=utf-8` but got "+response.headers.get('Content-Type'))
      }
    } else {
      throw new HttpError(response);
    }
    document.getElementById('remoteSessionDescription').value = JSON.stringify(json.SD)
    return json.SD
  }
  catch (e) {
    log(e);
  }
}

function handleTrack(event){
  var el = document.getElementById("video1")
  el.srcObject = event.streams[0]
  el.autoplay = true
  el.controls = true
}

// Set the handler for ICE connection state
// This will notify you when the peer has connected/disconnected
function handleICEConnectionStateChange(event){
  log("ICEConnectionStateChange: "+pc.iceConnectionState)
};

// pc.onnegotiationneeded = handleNegotiationNeeded;
// function handleNegotiationNeeded(){
// };

function handleGetUserMediaError(e) {
  switch(e.name) {
    case "NotFoundError":
      log("Unable to open your call because no camera and/or microphone" +
            "were found.");
      break;
    case "SecurityError":
    case "PermissionDeniedError":
      // Do nothing; this is the same as the user canceling the call.
      break;
    default:
      log("Error opening your camera and/or microphone: " + e.message);
      break;
  }
}

class HttpError extends Error {
  constructor(response) {
    super(`${response.status} for ${response.url}`);
    this.name = 'HttpError';
    this.response = response;
  }
}

var log = msg => {
  document.getElementById('logs').innerHTML += msg + '<br>'
}

window.addEventListener('unhandledrejection', function(event) {
  // the event object has two special properties:
  // alert(event.promise); // [object Promise] - the promise that generated the error
  // alert(event.reason); // Error: Whoops! - the unhandled error object
  alert("Event: "+event.promise+". Reason: "+event.reason); 
});
