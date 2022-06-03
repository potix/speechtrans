package message

const (
	MTypePing           string = "ping"
	MTypeInAudioConfReq        = "inAudioConfReq"
	MTypeInAudioDataReq        = "inAudioDataReq"
)

const (
	EncodingWave string = "wave"
)

type Message struct {
	MType       string
	InAudioConf *InAudioConf
	InAudioData *InAudioData
}

type InAudioConf struct {
	Encoding     string
	SampleRate   int
	SampleSize   int
	ChannelCount int
}

type InAudioData struct {
	DataBytes    []byte
}
