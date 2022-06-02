
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
	if (audioInputDevicesVue.selectedVideoInputDevice == "" ||
	    audioOutputDevicesVue.selectedAudioInputDevice == "") {
		return
	}
	console.log(audioInputDevicesVue.selectedVideoInputDevice);
	console.log(audioOutputDevicesVue.selectedAudioInputDevice);
	// XXXX startSignaling()
	navigator.mediaDevices.getUserMedia({
		audio: { deviceId: audioInputDevicesVue.selectedAudioInputDevice }
	}).then(function(stream) {
		console.log(stream);
        })
        .catch(function(err) {
                console.log("in startLocalVideo: " + err.name + ": " + err.message);
        });
}



