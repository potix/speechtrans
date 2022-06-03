//class AudioWorkletProcessor {}
//function registerProcessor(a, b) {
//}

class RecorderWorkletProcessor extends AudioWorkletProcessor {
        constructor(args) {
                super(args);
		this.port.onmessage = (event) => {
                        const params = JSON.parse(event.data);
			this.sampleRate = params.sampleRate
			this.sampleSize = params.sampleSize
			this.channelCount = params.channelCount
			this.streamNodeChannelCount = params.streamNodeChannelCount
			console.log(this.sampleRate)
			console.log(this.sampleSize)
			console.log(this.channelCount)
			console.log(this.streamNodeChannelCount)
                };
        }
	convertRawToWave(rawValue, min, max, unsigned) {
		let waveValue = 0
		if (unsigned) {
			rawValue = (rawValue + 1)
		}
		if (rawValue < 0) {
			waveValue = rawValue * -1 * min 
		} else {
			waveValue = rawValue * max
			if (unsigned) {
				waveValue = waveValue / 2
			}
		}
		return Math.floor(waveValue)
	}
	mixWave(waveValue1, waveValue2, min, max) {
		mixWaveValue = waveValue1 + waveValue2
		if (mixWaveValue < min) {
			mixWaveValue = min
		} else if (mixWaveValue > max) {
			mixWaveValue = max
		}
		return mixWaveValue
	}
	i24bitTobytes(value) {
		let hv = (value >> 16) & 0xff
		if (value < 0) {
			hv |= 0x80
		} 
		return [ hv & 0xff, (value >> 8) & 0xff, value & 0xff ]
	}
	i16bitTobytes(value) {
		let hv = (value >> 8) & 0xff
		if (value < 0) {
			hv |= 0x80
		}
		return [ hv, value & 0xff ]
	}
	toBytes(waveValues) {
		let uint8Array = new Uint8Array(waveValues.length * this.sampleSize / 8)
		let idx = 0
		for (const waveValue of waveValues) {
			if (this.sampleSize == 8) {
				uint8Array[idx++] = waveValue
			} else if (this.sampleSize == 16) {
				const bytes = this.i16bitTobytes(waveValue)
				uint8Array[idx++] = bytes[1]
				uint8Array[idx++] = bytes[0]
			} else if (this.sampleSize == 24) {
				const bytes = this.i24bitTobytes(waveValue)
				uint8Array[idx++] = bytes[2]
				uint8Array[idx++] = bytes[1]
				uint8Array[idx++] = bytes[0]
			}
		}
		return Array.from(uint8Array)
	}
        process(inputs, outputs, parameters) {
		//console.log(inputs)
		let minValue = 0
		let maxValue = 255
		let unsigned = true
		if (this.sampleSize == 16) {
			minValue = -32768;
			maxValue = 32767;
			unsigned = false
		} else if (this.sampleSize == 24) {
			minValue = -8388608;
			maxValue = 8388607;
			unsigned = false
		}
		let waveValues = []
		for (const idx1 in inputs) {
			let inChannelsLen = 0
			let inChannelDataLen = 0
			const inChannels = inputs[idx1]
			inChannelsLen = inChannels.length
			// * チャンネルが4つとかある時どうすればいいか分からない */
			if (inChannelsLen > this.channelCount) {
				inChannelsLen = this.channelCount
			}
			for (const idx2 in inChannels) {
				let inChannelData = inChannels[idx2]
				inChannelDataLen = inChannelData.length
				break
			}
			for (let idx3 = 0, idx4 = 0; idx3 < inChannelDataLen; idx3 += 1, idx4 += inChannelsLen) {
				for (let idx2 = 0; idx2 < inChannelsLen; idx2 += 1) {
					if (waveValues[idx4 + idx2]) {
						newWaveValue = this.convertRawToWave(inputs[idx1][idx2][idx3], minValue, maxValue, unsigned)
						waveValues[idx4 + idx2] = this.mixWave(wavData[idx3], newWaveValue, minValue, maxValue)
					} else {
						waveValues[idx4 + idx2] = this.convertRawToWave(inputs[idx1][idx2][idx3], minValue, maxValue, unsigned)
					}
				}
			}
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
		let message = { MType: "inAudio", waveBytes: this.toBytes(waveValues) };
		let jsonMessage = JSON.stringify(message);
		this.port.postMessage(jsonMessage);
                return true;
        }
}
registerProcessor('recorder-worklet', RecorderWorkletProcessor);
