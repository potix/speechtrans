let inputAudioContext = null
let wsSocket = null

let audioInputDevicesVue = new Vue({
	el: '#div_for_audio_input_devices',
	data: {
		selectedAudioInputDevice: 'default',
		audioInputDevices: [],
	},
	mounted : function(){
	},
	methods: {
	}
});

let audioOutputDevicesVue = new Vue({
	el: '#div_for_audio_output_devices',
	data: {
		selectedAudioOutputDevice: 'default',
		audioOutputDevices: [],
	},
	mounted : function(){
		this.setSkinId();
	},
	methods: {
		setSkinId: function() {
			if (this.selectedAudioOutputDevice == "") {
				return
			}
			console.log("setSkinId:" + this.selectedAudioOutputDevice);
			const outputAudio = document.getElementById('output_audio');
			if (outputAudio.setSinkId) {
				outputAudio.setSinkId(this.selectedAudioOutputDevice)
				.then(function(stream) {
					console.log("done set skinId");
				})
				.catch(function(err) {
					console.log("in setSkinId: " + err.name + ": " + err.message);
				});
			} else {
				// [ firefox ]
				// about:config
				// media.setsinkid.enabled
				// change to true
				console.log("can not set skinId");
			}
		},
	}
});

window.onload = function() {
	tryGetUserMedia();
}

function tryGetUserMedia() {
	// getUserMediaを実行してユーザーにデバイスの利用許可を促す
	// 許可されるとデバイス一覧が取得できるようになる
	navigator.mediaDevices.getUserMedia({video: false, audio: true })
	.then(function(stream) {
		getAudioInOutDevices();
	})
	.catch(function(err) {
		console.log("in tryGetUserMedia: " + err.name + ": " + err.message);
	});
}

function getAudioInOutDevices() {
    if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) {
        console.log("enumerateDevices() not supported.");
    }
    navigator.mediaDevices.enumerateDevices()
    .then(function(devices) {
        let audioInputDevices = [];
        let audioOutputDevices = [];
        devices.forEach(function(device) {
            if (device.kind == "audioinput") {
	        audioInputDevices.push({ "deviceId" : device.deviceId,  "label" : device.label});
	    }
            if (device.kind == "audiooutput") {
	        audioOutputDevices.push({ "deviceId" : device.deviceId,  "label" : device.label});
	    }
        });
	console.log(audioInputDevices);
	console.log(audioOutputDevices);
	audioInputDevicesVue.audioInputDevices = audioInputDevices;
	audioOutputDevicesVue.audioOutputDevices = audioOutputDevices;
    })
    .catch(function(err) {
        console.log(err.name + ": " + err.message);
    });
}

function startRecording() {
	if (audioInputDevicesVue.selectedAudioInputDevice == "" ||
	    audioOutputDevicesVue.selectedAudioOutputDevice == "") {
		return
	}
	console.log(audioInputDevicesVue.selectedAudioInputDevice);
	console.log(audioOutputDevicesVue.selectedAudioOutputDevice);
	navigator.mediaDevices.getUserMedia({
		audio: { deviceId: audioInputDevicesVue.selectedAudioInputDevice,
			 sampleRate: 96000,
			 sampleSize: 24,
			 channelCount: 2,
			 autoGainControl: true,
			 noiseSuppression: true,
			 echoCancellation: true }
	}).then(function(stream) {
		connectWebsocket(stream);
        })
        .catch(function(err) {
                console.log("in startRecording: " + err.name + ": " + err.message);
        });
	const startLamp = document.getElementById('start_lamp');
	startLamp.setAttribute("class", "border-radius background-color-red inline-block" )
}

function stopRecording() {
	if (inputAudioContext) {
		inputAudioContext.close();
		inputAudioContext = null;
	}
	if (wsSocket) {
		wsSocket.close();
		wsSocket = null;
	}
	const startLamp = document.getElementById('start_lamp');
	startLamp.setAttribute("class", "border-radius background-color-gray inline-block" )
}

function connectWebsocket(stream) {
	wsSocket = new WebSocket("wss://" + location.host + location.pathname + "ws/trans", "translation");
	wsSocket.onopen = event => {
		console.log("signaling open");
		connectWorkletNode(stream, wsSocket)
	};
	wsSocket.onmessage = event => {
		console.log("signaling message");
	}
	wsSocket.onerror = event => {
		console.log("signaling error");
		console.log(event);
	}
	wsSocket.onclose = event => {
		console.log("signaling close");
		console.log(event);
	}
}

function connectWorkletNode(stream, wsSocket) {
	inputAudioContext = new AudioContext();
	inputAudioContext.audioWorklet.addModule('js/recorder_worklet.js').then(function () {
		const recorder = new AudioWorkletNode(inputAudioContext, 'recorder-worklet');
		recorder.port.onmessage = (event) => {
			    sendRawData(wsSocket, event);
		};
		const audioInput = inputAudioContext.createMediaStreamSource(stream);
                audioInput.connect(recorder);
                recorder.connect(inputAudioContext.destination);
        });
}

function sendRawData(wsSocket, event) {
	console.log(event.data);
}


