const SAMPLE_RATE=48000
const SAMPLE_SIZE=24
const CHANNEL_COUNT=1
let inputAudioContext = null
let wsSocket = null
let lastWaveBytes = []

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
	lastWaveBytes = []
	console.log(audioInputDevicesVue.selectedAudioInputDevice);
	console.log(audioOutputDevicesVue.selectedAudioOutputDevice);
	navigator.mediaDevices.getUserMedia({
		audio: { deviceId: audioInputDevicesVue.selectedAudioInputDevice,
			 sampleRate: SAMPLE_RATE,
			 sampleSize: SAMPLE_SIZE,
			 channelCount: CHANNEL_COUNT,
			 autoGainControl: false,
			 noiseSuppression: false,
			 echoCancellation: false }
	}).then(function(stream) {
		connectWebsocket(stream);
        })
        .catch(function(err) {
                console.log("in startRecording: " + err.name + ": " + err.message);
        });
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
	createWaveFile()
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
	inputAudioContext = new AudioContext({ sampleRate: SAMPLE_RATE });
	inputAudioContext.audioWorklet.addModule('js/recorder_worklet.js').then(function () {
		const audioInput = inputAudioContext.createMediaStreamSource(stream);
		const recorder = new AudioWorkletNode(inputAudioContext, 'recorder-worklet');
		const params = {
                         sampleRate: SAMPLE_RATE,
                         sampleSize: SAMPLE_SIZE,
                         channelCount: CHANNEL_COUNT,
                         streamNodeChannelCount: audioInput.channelCount
		}
		recorder.port.postMessage(JSON.stringify(params));
		recorder.port.onmessage = (event) => {
			    sendRawData(wsSocket, event);
		};
                audioInput.connect(recorder);
                recorder.connect(inputAudioContext.destination);
		const startLamp = document.getElementById('start_lamp');
		startLamp.setAttribute("class", "border-radius background-color-red inline-block" )
        });
}

function sendRawData(wsSocket, event) {
	const message = JSON.parse(event.data)
	lastWaveBytes =	lastWaveBytes.concat(message.waveBytes)
}

function createWaveFile() {
	let arrayBuffer = new ArrayBuffer(lastWaveBytes.length + 4 + 4 + 4 + 4 + 4 + 2 + 2 + 4 + 4 + 2 + 2 + 4 + 4)
	let dataView = new DataView(arrayBuffer);
	let offset = 0
	dataView.setUint8(offset++, 0x52)
	dataView.setUint8(offset++, 0x49)
	dataView.setUint8(offset++, 0x46)
	dataView.setUint8(offset++, 0x46)

	dataView.setUint32(offset, arrayBuffer.byteLength - 8, true)
	offset += 4

	dataView.setUint8(offset++, 0x57)
	dataView.setUint8(offset++, 0x41)
	dataView.setUint8(offset++, 0x56)
	dataView.setUint8(offset++, 0x45)

	dataView.setUint8(offset++, 0x66)
	dataView.setUint8(offset++, 0x6d)
	dataView.setUint8(offset++, 0x74)
	dataView.setUint8(offset++, 0x20)

	dataView.setUint32(offset, 16, true)
	offset += 4

	dataView.setUint16(offset, 1, true)
	offset += 2

	dataView.setUint16(offset, CHANNEL_COUNT, true)
	offset += 2

	dataView.setUint32(offset, SAMPLE_RATE, true)
	offset += 4

	dataView.setUint32(offset, SAMPLE_RATE * (SAMPLE_SIZE / 8) * CHANNEL_COUNT, true)
	offset += 4

	dataView.setUint16(offset,  (SAMPLE_SIZE / 8) * CHANNEL_COUNT, true)
	offset += 2

	dataView.setUint16(offset,  SAMPLE_SIZE, true)
	offset += 2

	dataView.setUint8(offset++, 0x64)
	dataView.setUint8(offset++, 0x61)
	dataView.setUint8(offset++, 0x74)
	dataView.setUint8(offset++, 0x61)

	dataView.setUint32(offset, lastWaveBytes.length, true)
	offset += 4

	for (let waveByte of lastWaveBytes) {
		dataView.setUint8(offset++, waveByte)
	}

	const url = URL.createObjectURL(new Blob([arrayBuffer], {type: "audio/wav"}))
	const link = document.getElementById('recoarded_audio');
	link.href = url;
	link.innerText = 'latest recoarded audio file';
}


