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
			waveValue = rawValue * min 
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
		if (mixWaveValue < min * -1) {
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
        process(inputs, outputs, parameters) {
		//console.log(inputs)
		let minValue = 0
		let maxValue = 255
		let unsigned = true
		if (this.sampleSize == 16) {
			minValue = 32768;
			maxValue = 32767;
			unsigned = false
		} else if (this.sampleSize == 24) {
			minValue = 8388608;
			maxValue = 8388607;
			unsigned = false
		}
		let waveArrayBuffer = null
		let waveDataView = null
		let waveArrayBufferOffset = 0
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
				const inChannelData = inChannels[idx2]
				inChannelDataLen = inChannelData.length
				break
			}
			if (waveArrayBuffer == null) { 
				const waveArrayBufferLen = inChannelsLen * inChannelDataLen * (this.sampleSize / 8)
				waveArrayBuffer = new ArrayBuffer(waveArrayBufferLen)
				waveDataView = new DataView(waveArrayBuffer)
			} else {
				waveArrayBufferOffset = 0
			}
			for (let idx3 = 0, idx4 = 0; idx3 < inChannelDataLen; idx3 += 1, idx4 += inChannelsLen) {
				for (let idx2 = 0; idx2 < inChannelsLen; idx2 += 1) {
					if (waveValues[idx4 + idx2]) {
						const newWaveValue = this.convertRawToWave(inputs[idx1][idx2][idx3], minValue, maxValue, unsigned)
						waveValues[idx4 + idx2] = this.mixWave(waveValues[idx4 + idx2], newWaveValue, minValue, maxValue)
					} else {
						waveValues[idx4 + idx2] = this.convertRawToWave(inputs[idx1][idx2][idx3], minValue, maxValue, unsigned)
					}
					if (this.sampleSize == 8) {
						waveDataView.setUint8(waveArrayBufferOffset++, waveValues[idx4 + idx2]) 
					} else if (this.sampleSize == 16) {
						const bytes = this.i16bitTobytes(waveValues[idx4 + idx2])
						waveDataView.setUint8(waveArrayBufferOffset++, bytes[1]) 
						waveDataView.setUint8(waveArrayBufferOffset++, bytes[0]) 
					} else if (this.sampleSize == 24) {
						const bytes = this.i24bitTobytes(waveValues[idx4 + idx2])
						waveDataView.setUint8(waveArrayBufferOffset++, bytes[2]) 
						waveDataView.setUint8(waveArrayBufferOffset++, bytes[1]) 
						waveDataView.setUint8(waveArrayBufferOffset++, bytes[0]) 
					}
				}
			}
		}
		let message = { MType: "inAudio", waveBytes: Array.from(new Uint8Array(waveArrayBuffer)) };
		let jsonMessage = JSON.stringify(message);
		this.port.postMessage(jsonMessage);
                return true;
        }
}
registerProcessor('recorder-worklet', RecorderWorkletProcessor);
