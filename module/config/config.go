package config

var (
	// 导出变量（首字母大写）
	KeyboardInputTime   int64         // 上次发送 typing 的时间
	ContinueNextMessage bool          // 是否继续发送下一条消息（如 pet_status 相关）
	WaitTime            int64 = 15    // 每次循环等待时间（秒）
	MessageBackground   bool  = false // 消息背景开关

	MusicTitle   string
	MusicArtist  string
	MusicPlaying bool
	Phone        PhoneStatus

	TopApplicationTitle string

	IotPort       int = 51851
	VRChatOSCPort int = 9000
)

type PhoneStatus struct {
	Screen     bool
	AppName    string
	AppPackage string
	Battery    int
	Power      bool
}
