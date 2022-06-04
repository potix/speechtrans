package message

const (
	MTypePing           string = "ping"               // server <-----> client 

	MTypeInAudioConfReq        = "inAudioConfReq"     // server <-----  client
	MTypeInAudioConfRes        = "inAudioConfRes"     // server  -----> client
	MTypeInAudioDataReq        = "inAudioDataReq"     // server <-----  client
	MTypeInAudioDataRes        = "inAudioDataRes"     // server  -----> client
	MTypeInAudioDataEndReq     = "inAudioDataEndReq"  // server <-----  clinet
	MTypeInAudioDataEndRes     = "inAudioDataEndRes"  // server  -----> clinet

        MTypeToTextNotify          = "toTextNotify"       // server  -----> client

	MTypeTranslateReq          = "translateReq"       // server <-----  clinet
	MTypeTranslateRes          = "translateRes"       // server  -----> clinet
)

const (
	EncodingWave    string  = "wave"
	EncodingMp3     string  = "mp3"
	EncodingOggOpus string  = "oggOpus"
)

type Message struct {
	MType         string
	Error         *Error
	InAudioConf   *InAudioConf
	InAudioData   *InAudioData
	TransConf     *TransConf
	TransResult   *TransResult
}

type Error struct {
	Message    string
}

type InAudioConf struct {
	Encoding     string
	SampleRate   int32
	SampleSize   int32
	ChannelCount int32
	SrcLang      string
}

type InAudioData struct {
	DataBytes []byte
	NormMin   float64
	NormMax   float64
}

type OutAudio struct {
	Encoding string
	DataBytes []byte
}

type TransConf struct {
	SrcLang      string
	DstLang      string
	Gender       string
}

type TransResult struct {
	Encoding string
	DataBytes []byte
}
