const SAMPLE_RATE=48000
const SAMPLE_SIZE=16
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

let noiseSupressionVue = new Vue({
	el: '#noise-suppression',
	data: {
		checked: true
	},
	mounted : function(){
	},
	methods: {
	}
});

let inputLanguagesVue = new Vue({
	el: '#div_for_input_languages',
	data: {
		selectedInputLanguage: 'ja-JP',
	},
	mounted : function(){
	},
	methods: {
	}
});

let outputLanguagesVue = new Vue({
	el: '#div_for_output_languages',
	data: {
		selectedOutputLanguage: 'en-US',
	},
	mounted : function(){
	},
	methods: {
	}
});

let outputGenderVue = new Vue({
	el: '#div_for_output_gender',
	data: {
		selectedOutputGender: 'female',
	},
	mounted : function(){
	},
	methods: {
	}
});

window.onload = function() {
	tryGetUserMedia();
	connectWebsocket()
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

function connectWebsocket() {
	wsSocket = new WebSocket("wss://" + location.host + location.pathname + "ws/trans", "translation");
	wsSocket.onopen = event => {
		console.log("websocket open");
	};
	wsSocket.onmessage = event => {
		console.log("webcosket message");
		let msg = JSON.parse(event.data);
		if (msg.MType == "ping") {
			// nothig to do
		} else if (msg.MType == "inAudioConfRes") {
			if (msg.Error && msg.Error.Message != "" ) {
				console.log("error in inAudioConfRes: " + msg.Error.Message)
			}
		} else if (msg.MType == "inAudioDataRes") {
			if (msg.Error && msg.Error.Message != "" ) {
				console.log("error in inAudioDataRes: " + msg.Error.Message)
			}
		} else if (msg.MType == "inAudioDataEndRes") {
			if (msg.Error && msg.Error.Message != "" ) {
				console.log("error in inAudioDataEndRes: " + msg.Error.Message)
			}
		}
	}
	wsSocket.onerror = event => {
		console.log("websocket error");
		console.log(event);
		wsSocket.close();
		wsSocket = null;
		setTimeout(function(){ connectWebsocket() }, 2000);
	}
	wsSocket.onclose = event => {
		console.log("websocket close");
		console.log(event);
		wsSocket.close();
		wsSocket = null;
		setTimeout(function(){ connectWebsocket() }, 2000);
	}
}

function startRecording() {
	if (audioInputDevicesVue.selectedAudioInputDevice == "" ||
	    audioOutputDevicesVue.selectedAudioOutputDevice == "") {
		window.alert("select input/output audio device");
		return
	}
	if (inputLanguagesVue.selectedInputLanguage == outputLanguagesVue.selectedOutputLanguage) {
		window.alert("Select different languages for input and output");
		return
	}
	if (!wsSocket || wsSocket.readyState != 1) {
		window.alert("no websocket connection");
		return
	}
	lastWaveBytes = []
	console.log(audioInputDevicesVue.selectedAudioInputDevice);
	console.log(audioOutputDevicesVue.selectedAudioOutputDevice);
	console.log(noiseSupressionVue.checked);
	navigator.mediaDevices.getUserMedia({
		audio: { deviceId: audioInputDevicesVue.selectedAudioInputDevice,
			 sampleRate: SAMPLE_RATE,
			 sampleSize: SAMPLE_SIZE,
			 channelCount: CHANNEL_COUNT,
			 noiseSuppression: noiseSupressionVue.checked,
			 autoGainControl: false,
			 echoCancellation: false }
	}).then(function(stream) {
		connectWorkletNode(stream)
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
	const startLamp = document.getElementById('start_lamp');
	startLamp.setAttribute("class", "border-radius background-color-gray inline-block" )
	const message = {
		MType: "inAudioDataEndReq",
	};
	wsSocket.send(JSON.stringify(message));
	createWaveFile()
}

function connectWorkletNode(stream) {
	inputAudioContext = new AudioContext({ sampleRate: SAMPLE_RATE });
	console.log(inputAudioContext.sampleRate)
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
			    sendRawData(event);
		};
                audioInput.connect(recorder);
                recorder.connect(inputAudioContext.destination);
		const startLamp = document.getElementById('start_lamp');
		startLamp.setAttribute("class", "border-radius background-color-red inline-block" )
        });
}

function sendRawData(event) {
	if (lastWaveBytes.length == 0) {
		const message = {
                        MType: "inAudioConfReq",
                        InAudioConf: {
                                Encoding:"wave",
                                SampleRate:SAMPLE_RATE,
                                SampleSize:SAMPLE_SIZE,
                                ChannelCount:CHANNEL_COUNT,
                                SrcLang: inputLanguagesVue.selectedInputLanguage,
                                DstLang: outputLanguagesVue.selectedOutputLanguage,
				Gender:  outputGenderVue.selectedOutputGender,
                        }
                };
		wsSocket.send(JSON.stringify(message));
	}
	wsSocket.send(event.data);
	const message = JSON.parse(event.data);
	lastWaveBytes =	lastWaveBytes.concat(message.InAudioData.DataBytes);
}

function createWaveFile() {
	let arrayBuffer = new ArrayBuffer(4 + 4 + 4 + 4 + 4 + 2 + 2 + 4 + 4 + 2 + 2 + 4 + 4 + lastWaveBytes.length)
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
	link.innerText = 'last recoarded audio';
}
