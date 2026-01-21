package nextui

// NextVal represents the NextUI system configuration loaded from nextval.elf.
// This structure matches the JSON output from the NextUI configuration utility.
type NextVal struct {
	Font            int    `json:"font"`
	Color1          string `json:"color1"`
	Color2          string `json:"color2"`
	Color3          string `json:"color3"`
	Color4          string `json:"color4"`
	Color5          string `json:"color5"`
	Color6          string `json:"color6"`
	BGColor         string `json:"bgcolor"`
	Radius          int    `json:"radius"`
	ShowClock       int    `json:"showclock"`
	Clock24h        int    `json:"clock24h"`
	BatteryPerc     int    `json:"batteryperc"`
	MenuAnim        int    `json:"menuanim"`
	MenuTransitions int    `json:"menutransitions"`
	Recents         int    `json:"recents"`
	GameArt         int    `json:"gameart"`
	ScreenTimeout   int    `json:"screentimeout"`
	SuspendTimeout  int    `json:"suspendTimeout"`
	SwitcherScale   int    `json:"switcherscale"`
	Haptics         int    `json:"haptics"`
	RomFolderBg     int    `json:"romfolderbg"`
	SaveFormat      int    `json:"saveFormat"`
	StateFormat     int    `json:"stateFormat"`
	MuteLeds        int    `json:"muteLeds"`
	ArtWidth        int    `json:"artWidth"`
	Wifi            int    `json:"wifi"`
	FontPath        string `json:"fontpath"`
}
