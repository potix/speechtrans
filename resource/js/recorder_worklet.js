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

		/* loopbeck */
		/*
		for (const idx1 in inputs) {
			let inChannels = inputs[idx1] 
			let outChannels = outputs[idx1]
			for (const idx2 in inChannels) {
				let inChannel = inChannels[idx2]
				let outChannel = outChannels[idx2]
				for (const idx3 in inChannel) {
					outChannel[idx3] = inChannel[idx3]
				}

			}
		}
		*/

		let message = { MType: "inAudio", RawData: newInputs };
		let jsonMessage = JSON.stringify(message);
		this.port.postMessage(jsonMessage);
                return true;
        }
}
registerProcessor('recorder-worklet', RecorderWorkletProcessor);
