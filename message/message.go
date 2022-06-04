package message

const (
	MTypePing           string = "ping"               // server <-----> client 

	MTypeInAudioConfReq        = "inAudioConfReq"     // server <-----  client
	MTypeInAudioConfRes        = "inAudioConfRes"     // server  -----> client
	MTypeInAudioDataReq        = "inAudioDataReq"     // server <-----  client
	MTypeInAudioDataRes        = "inAudioDataRes"     // server  -----> client
	MTypeInAudioDataEndReq     = "inAudioDataEndReq"  // server <-----  clinet
	MTypeInAudioDataEndRes     = "inAudioDataEndRes"  // server  -----> clinet

	MTypeOutAudioReq           = "outAudioReq"        // server  -----> clinet
	MTypeOutAudioRes           = "outAudioRes"        // server <-----  clinet
)

const (
	EncodingWave    string  = "wave"
	EncodingMp3     string  = "mp3"
	EncodingOggOpus string  = "oggOpus"
)

type Message struct {
	MType        string
	Error        *Error
	InAudioConf  *InAudioConf
	InAudioData  *InAudioData
	OutAudio     *OutAudio
}

type Error struct {
	Message    string
}

type InAudioConf struct {
	Encoding     string
	SampleRate   int
	SampleSize   int
	ChannelCount int
	SrcLang      string
	DstLang      string
	Gender       string
}

type InAudioData struct {
	DataBytes []byte
}

type OutAudio struct {
	Encoding string
	DataBytes []byte
}
