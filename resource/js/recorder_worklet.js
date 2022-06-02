class RecorderWorkletProcessor extends AudioWorkletProcessor {
        constructor(args) {
                super(args);
        }
        process(inputs, outputs, parameters) {
		let newInputs = []
		for (const channels of inputs) {
			let newChannels = []
			for (const channel of channels) {
				let newChannel = Array.from(channel)
				newChannels.push(newChannel);
			}
			newInputs.push(newChannels)

		}
		let message = { MType: "inAudio", RawData: newInputs };
		let jsonMessage = JSON.stringify(message);
		this.port.postMessage(jsonMessage);
                return true;
        }
}
registerProcessor('recorder-worklet', RecorderWorkletProcessor);
