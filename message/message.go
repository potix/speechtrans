package message

const (
	MTypePing           string = "ping"               // server <-----> client 

	MTypeInAudioConfReq        = "inAudioConfReq"     // server <-----  client
	MTypeInAudioConfRes        = "inAudioConfRes"     // server  -----> client
	MTypeInAudioDataReq        = "inAudioDataReq"     // server <-----  client
	MTypeInAudioDataRes        = "inAudioDataRes"     // server  -----> client
	MTypeInAudioDataEndReq     = "inAudioDataEndReq"  // server <-----  clinet
	MTypeInAudioDataEndRes     = "inAudioDataEndRes"  // server  -----> clinet

	MTypeOutAudioConfReq       = "outAudioConfReq"    // server  -----> clinet
	MTypeOutAudioConfRes       = "outAudioConfRes"    // server <-----  clinet
	MTypeOutAudioDataReq       = "outAudioDataReq"    // server  -----> clinet
	MTypeOutAudioDataRes       = "outAudioDataRes"    // server <-----  clinet
	MTypeOutAudioDataEndReq    = "outAudioDataEndReq" // server  -----> clinet
	MTypeOutAudioDataEndRes    = "outAudioDataEndRes" // server <-----  clinet
)

const (
	EncodingWave string = "wave"
)

type Message struct {
	MType       string
	Error       *Error
	InAudioConf *InAudioConf
	InAudioData *InAudioData
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

type OutAudioConf struct {
	Encoding string
}

type OutAudioData struct {
	DataBytes []byte
}
