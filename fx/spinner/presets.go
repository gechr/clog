package spinner

import "time"

// Predefined spinner frame sets adapted from https://github.com/sindresorhus/cli-spinners
// Pass any of these to [WithConfig] or the animation builder's Config method to change the animation.
var (
	Aesthetic = Config{
		Frames: []string{
			"▰▱▱▱▱▱▱",
			"▰▰▱▱▱▱▱",
			"▰▰▰▱▱▱▱",
			"▰▰▰▰▱▱▱",
			"▰▰▰▰▰▱▱",
			"▰▰▰▰▰▰▱",
			"▰▰▰▰▰▰▰",
			"▰▱▱▱▱▱▱",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Arc = Config{
		Frames:   []string{"◜", "◠", "◝", "◞", "◡", "◟"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Arrow2 = Config{
		Frames:   []string{"⬆️ ", "↗️ ", "➡️ ", "↘️ ", "⬇️ ", "↙️ ", "⬅️ ", "↖️ "},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Arrow3 = Config{
		Frames:   []string{"▹▹▹▹▹", "▸▹▹▹▹", "▹▸▹▹▹", "▹▹▸▹▹", "▹▹▹▸▹", "▹▹▹▹▸"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Balloon = Config{
		Frames:   []string{" ", ".", "o", "O", "@", "*", " "},
		Interval: 140 * time.Millisecond, //nolint:mnd // frame rate
	}
	Balloon2 = Config{
		Frames:   []string{".", "o", "O", "°", "O", "o", "."},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	BetaWave = Config{
		Frames: []string{
			"ρββββββ",
			"βρβββββ",
			"ββρββββ",
			"βββρβββ",
			"ββββρββ",
			"βββββρβ",
			"ββββββρ",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Binary = Config{
		Frames: []string{
			"010010",
			"001100",
			"100101",
			"111010",
			"111101",
			"010111",
			"101011",
			"111000",
			"110011",
			"110101",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	BluePulse = Config{
		Frames:   []string{"🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	BouncingBall = Config{
		Frames: []string{
			"( ●    )",
			"(  ●   )",
			"(   ●  )",
			"(    ● )",
			"(     ●)",
			"(    ● )",
			"(   ●  )",
			"(  ●   )",
			"( ●    )",
			"(●     )",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	BoxBounce = Config{
		Frames:   []string{"▖", "▘", "▝", "▗"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	BoxBounce2 = Config{
		Frames:   []string{"▌", "▀", "▐", "▄"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Christmas = Config{
		Frames:   []string{"🌲", "🎄"},
		Interval: 400 * time.Millisecond, //nolint:mnd // frame rate
	}
	Circle = Config{
		Frames:   []string{"◡", "⊙", "◠"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	CircleHalves = Config{
		Frames:   []string{"◐", "◓", "◑", "◒"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	CircleQuarters = Config{
		Frames:   []string{"◴", "◷", "◶", "◵"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dot = Config{
		Frames:   []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots = Config{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	DotsBounce = Config{
		Frames:   []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots11 = Config{
		Frames:   []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots12 = Config{
		Frames: []string{
			"⢀⠀",
			"⡀⠀",
			"⠄⠀",
			"⢂⠀",
			"⡂⠀",
			"⠅⠀",
			"⢃⠀",
			"⡃⠀",
			"⠍⠀",
			"⢋⠀",
			"⡋⠀",
			"⠍⠁",
			"⢋⠁",
			"⡋⠁",
			"⠍⠉",
			"⠋⠉",
			"⠋⠉",
			"⠉⠙",
			"⠉⠙",
			"⠉⠩",
			"⠈⢙",
			"⠈⡙",
			"⢈⠩",
			"⡀⢙",
			"⠄⡙",
			"⢂⠩",
			"⡂⢘",
			"⠅⡘",
			"⢃⠨",
			"⡃⢐",
			"⠍⡐",
			"⢋⠠",
			"⡋⢀",
			"⠍⡁",
			"⢋⠁",
			"⡋⠁",
			"⠍⠉",
			"⠋⠉",
			"⠋⠉",
			"⠉⠙",
			"⠉⠙",
			"⠉⠩",
			"⠈⢙",
			"⠈⡙",
			"⠈⠩",
			"⠀⢙",
			"⠀⡙",
			"⠀⠩",
			"⠀⢘",
			"⠀⡘",
			"⠀⠨",
			"⠀⢐",
			"⠀⡐",
			"⠀⠠",
			"⠀⢀",
			"⠀⡀",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots13 = Config{
		Frames:   []string{"⣼", "⣹", "⢻", "⠿", "⡟", "⣏", "⣧", "⣶"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots14 = Config{
		Frames: []string{
			"⠉⠉",
			"⠈⠙",
			"⠀⠹",
			"⠀⢸",
			"⠀⣰",
			"⢀⣠",
			"⣀⣀",
			"⣄⡀",
			"⣆⠀",
			"⡇⠀",
			"⠏⠀",
			"⠋⠁",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots3 = Config{
		Frames:   []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots4 = Config{
		Frames: []string{
			"⠄",
			"⠆",
			"⠇",
			"⠋",
			"⠙",
			"⠸",
			"⠰",
			"⠠",
			"⠰",
			"⠸",
			"⠙",
			"⠋",
			"⠇",
			"⠆",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots5 = Config{
		Frames: []string{
			"⠋",
			"⠙",
			"⠚",
			"⠒",
			"⠂",
			"⠂",
			"⠒",
			"⠲",
			"⠴",
			"⠦",
			"⠖",
			"⠒",
			"⠐",
			"⠐",
			"⠒",
			"⠓",
			"⠋",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots6 = Config{
		Frames: []string{
			"⠁",
			"⠉",
			"⠙",
			"⠚",
			"⠒",
			"⠂",
			"⠂",
			"⠒",
			"⠲",
			"⠴",
			"⠤",
			"⠄",
			"⠄",
			"⠤",
			"⠴",
			"⠲",
			"⠒",
			"⠂",
			"⠂",
			"⠒",
			"⠚",
			"⠙",
			"⠉",
			"⠁",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots7 = Config{
		Frames: []string{
			"⠈",
			"⠉",
			"⠋",
			"⠓",
			"⠒",
			"⠐",
			"⠐",
			"⠒",
			"⠖",
			"⠦",
			"⠤",
			"⠠",
			"⠠",
			"⠤",
			"⠦",
			"⠖",
			"⠒",
			"⠐",
			"⠐",
			"⠒",
			"⠓",
			"⠋",
			"⠉",
			"⠈",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots8 = Config{
		Frames: []string{
			"⠁",
			"⠁",
			"⠉",
			"⠙",
			"⠚",
			"⠒",
			"⠂",
			"⠂",
			"⠒",
			"⠲",
			"⠴",
			"⠤",
			"⠄",
			"⠄",
			"⠤",
			"⠠",
			"⠠",
			"⠤",
			"⠦",
			"⠖",
			"⠒",
			"⠐",
			"⠐",
			"⠒",
			"⠓",
			"⠋",
			"⠉",
			"⠈",
			"⠈",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots8Bit = Config{
		Frames: []string{
			"⠀",
			"⠁",
			"⠂",
			"⠃",
			"⠄",
			"⠅",
			"⠆",
			"⠇",
			"⡀",
			"⡁",
			"⡂",
			"⡃",
			"⡄",
			"⡅",
			"⡆",
			"⡇",
			"⠈",
			"⠉",
			"⠊",
			"⠋",
			"⠌",
			"⠍",
			"⠎",
			"⠏",
			"⡈",
			"⡉",
			"⡊",
			"⡋",
			"⡌",
			"⡍",
			"⡎",
			"⡏",
			"⠐",
			"⠑",
			"⠒",
			"⠓",
			"⠔",
			"⠕",
			"⠖",
			"⠗",
			"⡐",
			"⡑",
			"⡒",
			"⡓",
			"⡔",
			"⡕",
			"⡖",
			"⡗",
			"⠘",
			"⠙",
			"⠚",
			"⠛",
			"⠜",
			"⠝",
			"⠞",
			"⠟",
			"⡘",
			"⡙",
			"⡚",
			"⡛",
			"⡜",
			"⡝",
			"⡞",
			"⡟",
			"⠠",
			"⠡",
			"⠢",
			"⠣",
			"⠤",
			"⠥",
			"⠦",
			"⠧",
			"⡠",
			"⡡",
			"⡢",
			"⡣",
			"⡤",
			"⡥",
			"⡦",
			"⡧",
			"⠨",
			"⠩",
			"⠪",
			"⠫",
			"⠬",
			"⠭",
			"⠮",
			"⠯",
			"⡨",
			"⡩",
			"⡪",
			"⡫",
			"⡬",
			"⡭",
			"⡮",
			"⡯",
			"⠰",
			"⠱",
			"⠲",
			"⠳",
			"⠴",
			"⠵",
			"⠶",
			"⠷",
			"⡰",
			"⡱",
			"⡲",
			"⡳",
			"⡴",
			"⡵",
			"⡶",
			"⡷",
			"⠸",
			"⠹",
			"⠺",
			"⠻",
			"⠼",
			"⠽",
			"⠾",
			"⠿",
			"⡸",
			"⡹",
			"⡺",
			"⡻",
			"⡼",
			"⡽",
			"⡾",
			"⡿",
			"⢀",
			"⢁",
			"⢂",
			"⢃",
			"⢄",
			"⢅",
			"⢆",
			"⢇",
			"⣀",
			"⣁",
			"⣂",
			"⣃",
			"⣄",
			"⣅",
			"⣆",
			"⣇",
			"⢈",
			"⢉",
			"⢊",
			"⢋",
			"⢌",
			"⢍",
			"⢎",
			"⢏",
			"⣈",
			"⣉",
			"⣊",
			"⣋",
			"⣌",
			"⣍",
			"⣎",
			"⣏",
			"⢐",
			"⢑",
			"⢒",
			"⢓",
			"⢔",
			"⢕",
			"⢖",
			"⢗",
			"⣐",
			"⣑",
			"⣒",
			"⣓",
			"⣔",
			"⣕",
			"⣖",
			"⣗",
			"⢘",
			"⢙",
			"⢚",
			"⢛",
			"⢜",
			"⢝",
			"⢞",
			"⢟",
			"⣘",
			"⣙",
			"⣚",
			"⣛",
			"⣜",
			"⣝",
			"⣞",
			"⣟",
			"⢠",
			"⢡",
			"⢢",
			"⢣",
			"⢤",
			"⢥",
			"⢦",
			"⢧",
			"⣠",
			"⣡",
			"⣢",
			"⣣",
			"⣤",
			"⣥",
			"⣦",
			"⣧",
			"⢨",
			"⢩",
			"⢪",
			"⢫",
			"⢬",
			"⢭",
			"⢮",
			"⢯",
			"⣨",
			"⣩",
			"⣪",
			"⣫",
			"⣬",
			"⣭",
			"⣮",
			"⣯",
			"⢰",
			"⢱",
			"⢲",
			"⢳",
			"⢴",
			"⢵",
			"⢶",
			"⢷",
			"⣰",
			"⣱",
			"⣲",
			"⣳",
			"⣴",
			"⣵",
			"⣶",
			"⣷",
			"⢸",
			"⢹",
			"⢺",
			"⢻",
			"⢼",
			"⢽",
			"⢾",
			"⢿",
			"⣸",
			"⣹",
			"⣺",
			"⣻",
			"⣼",
			"⣽",
			"⣾",
			"⣿",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots9 = Config{
		Frames:   []string{"⢹", "⢺", "⢼", "⣸", "⣇", "⡧", "⡗", "⡏"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	DotsCircle = Config{
		Frames:   []string{"⢎ ", "⠎⠁", "⠊⠑", "⠈⠱", " ⡱", "⢀⡰", "⢄⡠", "⢆⡀"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dqpb = Config{
		Frames:   []string{"d", "q", "p", "b"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	DwarfFortress = Config{
		Frames: []string{
			" ██████£££  ",
			"☺██████£££  ",
			"☺██████£££  ",
			"☺▓█████£££  ",
			"☺▓█████£££  ",
			"☺▒█████£££  ",
			"☺▒█████£££  ",
			"☺░█████£££  ",
			"☺░█████£££  ",
			"☺ █████£££  ",
			" ☺█████£££  ",
			" ☺█████£££  ",
			" ☺▓████£££  ",
			" ☺▓████£££  ",
			" ☺▒████£££  ",
			" ☺▒████£££  ",
			" ☺░████£££  ",
			" ☺░████£££  ",
			" ☺ ████£££  ",
			"  ☺████£££  ",
			"  ☺████£££  ",
			"  ☺▓███£££  ",
			"  ☺▓███£££  ",
			"  ☺▒███£££  ",
			"  ☺▒███£££  ",
			"  ☺░███£££  ",
			"  ☺░███£££  ",
			"  ☺ ███£££  ",
			"   ☺███£££  ",
			"   ☺███£££  ",
			"   ☺▓██£££  ",
			"   ☺▓██£££  ",
			"   ☺▒██£££  ",
			"   ☺▒██£££  ",
			"   ☺░██£££  ",
			"   ☺░██£££  ",
			"   ☺ ██£££  ",
			"    ☺██£££  ",
			"    ☺██£££  ",
			"    ☺▓█£££  ",
			"    ☺▓█£££  ",
			"    ☺▒█£££  ",
			"    ☺▒█£££  ",
			"    ☺░█£££  ",
			"    ☺░█£££  ",
			"    ☺ █£££  ",
			"     ☺█£££  ",
			"     ☺█£££  ",
			"     ☺▓£££  ",
			"     ☺▓£££  ",
			"     ☺▒£££  ",
			"     ☺▒£££  ",
			"     ☺░£££  ",
			"     ☺░£££  ",
			"     ☺ £££  ",
			"      ☺£££  ",
			"      ☺£££  ",
			"      ☺▓££  ",
			"      ☺▓££  ",
			"      ☺▒££  ",
			"      ☺▒££  ",
			"      ☺░££  ",
			"      ☺░££  ",
			"      ☺ ££  ",
			"       ☺££  ",
			"       ☺££  ",
			"       ☺▓£  ",
			"       ☺▓£  ",
			"       ☺▒£  ",
			"       ☺▒£  ",
			"       ☺░£  ",
			"       ☺░£  ",
			"       ☺ £  ",
			"        ☺£  ",
			"        ☺£  ",
			"        ☺▓  ",
			"        ☺▓  ",
			"        ☺▒  ",
			"        ☺▒  ",
			"        ☺░  ",
			"        ☺░  ",
			"        ☺   ",
			"        ☺  &",
			"        ☺ ☼&",
			"       ☺ ☼ &",
			"       ☺☼  &",
			"      ☺☼  & ",
			"      ‼   & ",
			"     ☺   &  ",
			"    ‼    &  ",
			"   ☺    &   ",
			"  ‼     &   ",
			" ☺     &    ",
			"‼      &    ",
			"      &     ",
			"      &     ",
			"     &   ░  ",
			"     &   ▒  ",
			"    &    ▓  ",
			"    &    £  ",
			"   &    ░£  ",
			"   &    ▒£  ",
			"  &     ▓£  ",
			"  &     ££  ",
			" &     ░££  ",
			" &     ▒££  ",
			"&      ▓££  ",
			"&      £££  ",
			"      ░£££  ",
			"      ▒£££  ",
			"      ▓£££  ",
			"      █£££  ",
			"     ░█£££  ",
			"     ▒█£££  ",
			"     ▓█£££  ",
			"     ██£££  ",
			"    ░██£££  ",
			"    ▒██£££  ",
			"    ▓██£££  ",
			"    ███£££  ",
			"   ░███£££  ",
			"   ▒███£££  ",
			"   ▓███£££  ",
			"   ████£££  ",
			"  ░████£££  ",
			"  ▒████£££  ",
			"  ▓████£££  ",
			"  █████£££  ",
			" ░█████£££  ",
			" ▒█████£££  ",
			" ▓█████£££  ",
			" ██████£££  ",
			" ██████£££  ",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Ellipsis = Config{
		Frames:   []string{"", ".", "..", "..."},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	FingerDance = Config{
		Frames:   []string{"🤘 ", "🤟 ", "🖖 ", "✋ ", "🤚 ", "👆 "},
		Interval: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	Fish = Config{
		Frames: []string{
			"~~~~~~~~~~~~~~~~~~~~",
			"> ~~~~~~~~~~~~~~~~~~",
			"º> ~~~~~~~~~~~~~~~~~",
			"(º> ~~~~~~~~~~~~~~~~",
			"((º> ~~~~~~~~~~~~~~~",
			"<((º> ~~~~~~~~~~~~~~",
			"><((º> ~~~~~~~~~~~~~",
			" ><((º> ~~~~~~~~~~~~",
			"~ ><((º> ~~~~~~~~~~~",
			"~~ <>((º> ~~~~~~~~~~",
			"~~~ ><((º> ~~~~~~~~~",
			"~~~~ <>((º> ~~~~~~~~",
			"~~~~~ ><((º> ~~~~~~~",
			"~~~~~~ <>((º> ~~~~~~",
			"~~~~~~~ ><((º> ~~~~~",
			"~~~~~~~~ <>((º> ~~~~",
			"~~~~~~~~~ ><((º> ~~~",
			"~~~~~~~~~~ <>((º> ~~",
			"~~~~~~~~~~~ ><((º> ~",
			"~~~~~~~~~~~~ <>((º> ",
			"~~~~~~~~~~~~~ ><((º>",
			"~~~~~~~~~~~~~~ <>((º",
			"~~~~~~~~~~~~~~~ ><((",
			"~~~~~~~~~~~~~~~~ <>(",
			"~~~~~~~~~~~~~~~~~ ><",
			"~~~~~~~~~~~~~~~~~~ <",
			"~~~~~~~~~~~~~~~~~~~~",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	FistBump = Config{
		Frames: []string{
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"\u3000🤜\u3000\u3000🤛\u3000 ",
			"\u3000\u3000🤜🤛\u3000\u3000 ",
			"\u3000🤜✨🤛\u3000\u3000 ",
			"🤜\u3000✨\u3000🤛\u3000 ",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Flip = Config{
		Frames: []string{
			"_",
			"_",
			"_",
			"-",
			"`",
			"`",
			"'",
			"´",
			"-",
			"_",
			"_",
			"_",
		},
		Interval: 70 * time.Millisecond, //nolint:mnd // frame rate
	}
	Globe = Config{
		Frames:   []string{"🌍", "🌎", "🌏"},
		Interval: 250 * time.Millisecond, //nolint:mnd // frame rate
	}
	Grenade = Config{
		Frames: []string{
			"،  ",
			"′  ",
			" ´ ",
			" ‾ ",
			"  ⸌",
			"  ⸊",
			"  |",
			"  ⁎",
			"  ⁕",
			" ෴ ",
			"  ⁓",
			"   ",
			"   ",
			"   ",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	GrowHorizontal = Config{
		Frames: []string{
			"▏",
			"▎",
			"▍",
			"▌",
			"▋",
			"▊",
			"▉",
			"▊",
			"▋",
			"▌",
			"▍",
			"▎",
		},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	GrowVertical = Config{
		Frames:   []string{"▁", "▃", "▄", "▅", "▆", "▇", "▆", "▅", "▄", "▃"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Hamburger = Config{
		Frames:   []string{"☱", "☲", "☴", "☲"},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	Jump = Config{
		Frames:   []string{"⢄", "⢂", "⢁", "⡁", "⡈", "⡐", "⡠"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Layer = Config{
		Frames:   []string{"-", "=", "≡"},
		Interval: 150 * time.Millisecond, //nolint:mnd // frame rate
	}
	Line = Config{
		Frames:   []string{"|", "/", "-", "\\"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Line2 = Config{
		Frames:   []string{"⠂", "-", "–", "-", "–", "-"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Material = Config{
		Frames: []string{
			"█▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"██▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"███▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"████▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"██████▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"██████▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"███████▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"████████▁▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"██████████▁▁▁▁▁▁▁▁▁▁",
			"███████████▁▁▁▁▁▁▁▁▁",
			"█████████████▁▁▁▁▁▁▁",
			"██████████████▁▁▁▁▁▁",
			"██████████████▁▁▁▁▁▁",
			"▁██████████████▁▁▁▁▁",
			"▁██████████████▁▁▁▁▁",
			"▁██████████████▁▁▁▁▁",
			"▁▁██████████████▁▁▁▁",
			"▁▁▁██████████████▁▁▁",
			"▁▁▁▁█████████████▁▁▁",
			"▁▁▁▁██████████████▁▁",
			"▁▁▁▁██████████████▁▁",
			"▁▁▁▁▁██████████████▁",
			"▁▁▁▁▁██████████████▁",
			"▁▁▁▁▁██████████████▁",
			"▁▁▁▁▁▁██████████████",
			"▁▁▁▁▁▁██████████████",
			"▁▁▁▁▁▁▁█████████████",
			"▁▁▁▁▁▁▁█████████████",
			"▁▁▁▁▁▁▁▁████████████",
			"▁▁▁▁▁▁▁▁████████████",
			"▁▁▁▁▁▁▁▁▁███████████",
			"▁▁▁▁▁▁▁▁▁███████████",
			"▁▁▁▁▁▁▁▁▁▁██████████",
			"▁▁▁▁▁▁▁▁▁▁██████████",
			"▁▁▁▁▁▁▁▁▁▁▁▁████████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁███████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁██████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█████",
			"█▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁████",
			"██▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁███",
			"██▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁███",
			"███▁▁▁▁▁▁▁▁▁▁▁▁▁▁███",
			"████▁▁▁▁▁▁▁▁▁▁▁▁▁▁██",
			"█████▁▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"█████▁▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"██████▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"████████▁▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"█████████▁▁▁▁▁▁▁▁▁▁▁",
			"███████████▁▁▁▁▁▁▁▁▁",
			"████████████▁▁▁▁▁▁▁▁",
			"████████████▁▁▁▁▁▁▁▁",
			"██████████████▁▁▁▁▁▁",
			"██████████████▁▁▁▁▁▁",
			"▁██████████████▁▁▁▁▁",
			"▁██████████████▁▁▁▁▁",
			"▁▁▁█████████████▁▁▁▁",
			"▁▁▁▁▁████████████▁▁▁",
			"▁▁▁▁▁████████████▁▁▁",
			"▁▁▁▁▁▁███████████▁▁▁",
			"▁▁▁▁▁▁▁▁█████████▁▁▁",
			"▁▁▁▁▁▁▁▁█████████▁▁▁",
			"▁▁▁▁▁▁▁▁▁█████████▁▁",
			"▁▁▁▁▁▁▁▁▁█████████▁▁",
			"▁▁▁▁▁▁▁▁▁▁█████████▁",
			"▁▁▁▁▁▁▁▁▁▁▁████████▁",
			"▁▁▁▁▁▁▁▁▁▁▁████████▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁███████▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁███████▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁███████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁███████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁████",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁███",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁███",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁██",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁██",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁██",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁█",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
			"▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁",
		},
		Interval: 17 * time.Millisecond, //nolint:mnd // frame rate
	}
	Meter = Config{
		Frames:   []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▰▰▱", "▰▱▱", "▱▱▱"},
		Interval: 143 * time.Millisecond, //nolint:mnd // frame rate
	}
	Mindblown = Config{
		Frames: []string{
			"😐 ",
			"😐 ",
			"😮 ",
			"😮 ",
			"😦 ",
			"😦 ",
			"😧 ",
			"😧 ",
			"🤯 ",
			"💥 ",
			"✨ ",
			"\u3000 ",
			"\u3000 ",
			"\u3000 ",
		},
		Interval: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	MiniDot = Config{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 83 * time.Millisecond, //nolint:mnd // frame rate
	}
	Monkey = Config{
		Frames:   []string{"🙈", "🙉", "🙊"},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	Moon = Config{
		Frames:   []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	Noise = Config{
		Frames:   []string{"▓", "▒", "░"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	OrangeBluePulse = Config{
		Frames:   []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 ", "🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	OrangePulse = Config{
		Frames:   []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Pipe = Config{
		Frames:   []string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Point = Config{
		Frames:   []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	Points = Config{
		Frames:   []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"},
		Interval: 143 * time.Millisecond, //nolint:mnd // frame rate
	}
	Pong = Config{
		Frames: []string{
			"▐⠂       ▌",
			"▐⠈       ▌",
			"▐ ⠂      ▌",
			"▐ ⠠      ▌",
			"▐  ⡀     ▌",
			"▐  ⠠     ▌",
			"▐   ⠂    ▌",
			"▐   ⠈    ▌",
			"▐    ⠂   ▌",
			"▐    ⠠   ▌",
			"▐     ⡀  ▌",
			"▐     ⠠  ▌",
			"▐      ⠂ ▌",
			"▐      ⠈ ▌",
			"▐       ⠂▌",
			"▐       ⠠▌",
			"▐       ⡀▌",
			"▐      ⠠ ▌",
			"▐      ⠂ ▌",
			"▐     ⠈  ▌",
			"▐     ⠂  ▌",
			"▐    ⠠   ▌",
			"▐    ⡀   ▌",
			"▐   ⠠    ▌",
			"▐   ⠂    ▌",
			"▐  ⠈     ▌",
			"▐  ⠂     ▌",
			"▐ ⠠      ▌",
			"▐ ⡀      ▌",
			"▐⠠       ▌",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Pulse = Config{
		Frames:   []string{"█", "▓", "▒", "░"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	RollingLine = Config{
		Frames:   []string{"/  ", " - ", " \\ ", "  |", "  |", " \\ ", " - ", "/  "},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Runner = Config{
		Frames:   []string{"🚶 ", "🏃 "},
		Interval: 140 * time.Millisecond, //nolint:mnd // frame rate
	}
	Sand = Config{
		Frames: []string{
			"⠁",
			"⠂",
			"⠄",
			"⡀",
			"⡈",
			"⡐",
			"⡠",
			"⣀",
			"⣁",
			"⣂",
			"⣄",
			"⣌",
			"⣔",
			"⣤",
			"⣥",
			"⣦",
			"⣮",
			"⣶",
			"⣷",
			"⣿",
			"⡿",
			"⠿",
			"⢟",
			"⠟",
			"⡛",
			"⠛",
			"⠫",
			"⢋",
			"⠋",
			"⠍",
			"⡉",
			"⠉",
			"⠑",
			"⠡",
			"⢁",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Shark = Config{
		Frames: []string{
			"▐|\\____________▌",
			"▐_|\\___________▌",
			"▐__|\\__________▌",
			"▐___|\\_________▌",
			"▐____|\\________▌",
			"▐_____|\\_______▌",
			"▐______|\\______▌",
			"▐_______|\\_____▌",
			"▐________|\\____▌",
			"▐_________|\\___▌",
			"▐__________|\\__▌",
			"▐___________|\\_▌",
			"▐____________|\\▌",
			"▐____________/|▌",
			"▐___________/|_▌",
			"▐__________/|__▌",
			"▐_________/|___▌",
			"▐________/|____▌",
			"▐_______/|_____▌",
			"▐______/|______▌",
			"▐_____/|_______▌",
			"▐____/|________▌",
			"▐___/|_________▌",
			"▐__/|__________▌",
			"▐_/|___________▌",
			"▐/|____________▌",
		},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SimpleDots = Config{
		Frames:   []string{".  ", ".. ", "...", "   "},
		Interval: 400 * time.Millisecond, //nolint:mnd // frame rate
	}
	SimpleDotsScrolling = Config{
		Frames:   []string{".  ", ".. ", "...", " ..", "  .", "   "},
		Interval: 200 * time.Millisecond, //nolint:mnd // frame rate
	}
	Smiley = Config{
		Frames:   []string{"😄 ", "😝 "},
		Interval: 200 * time.Millisecond, //nolint:mnd // frame rate
	}
	SoccerHeader = Config{
		Frames: []string{
			" 🧑⚽️       🧑 ",
			"🧑  ⚽️      🧑 ",
			"🧑   ⚽️     🧑 ",
			"🧑    ⚽️    🧑 ",
			"🧑     ⚽️   🧑 ",
			"🧑      ⚽️  🧑 ",
			"🧑       ⚽️🧑  ",
			"🧑      ⚽️  🧑 ",
			"🧑     ⚽️   🧑 ",
			"🧑    ⚽️    🧑 ",
			"🧑   ⚽️     🧑 ",
			"🧑  ⚽️      🧑 ",
		},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Speaker = Config{
		Frames:   []string{"🔈 ", "🔉 ", "🔊 ", "🔉 "},
		Interval: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	SquareCorners = Config{
		Frames:   []string{"◰", "◳", "◲", "◱"},
		Interval: 180 * time.Millisecond, //nolint:mnd // frame rate
	}
	Squish = Config{
		Frames:   []string{"╫", "╪"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Stars = Config{
		Frames: []string{
			"·",
			"✢",
			"✳",
			"✶",
			"✻",
			"✽",
			"✽",
			"✻",
			"✶",
			"✳",
			"✢",
			"·",
		},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	TimeTravel = Config{
		Frames: []string{
			"🕛 ",
			"🕚 ",
			"🕙 ",
			"🕘 ",
			"🕗 ",
			"🕖 ",
			"🕕 ",
			"🕔 ",
			"🕓 ",
			"🕒 ",
			"🕑 ",
			"🕐 ",
		},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle = Config{
		Frames:   []string{"⊶", "⊷"},
		Interval: 250 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle10 = Config{
		Frames:   []string{"㊂", "㊀", "㊁"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle11 = Config{
		Frames:   []string{"⧇", "⧆"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle12 = Config{
		Frames:   []string{"☗", "☖"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle13 = Config{
		Frames:   []string{"=", "*", "-"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle2 = Config{
		Frames:   []string{"▫", "▪"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle3 = Config{
		Frames:   []string{"□", "■"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle4 = Config{
		Frames:   []string{"■", "□", "▪", "▫"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle5 = Config{
		Frames:   []string{"▮", "▯"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle6 = Config{
		Frames:   []string{"ဝ", "၀"},
		Interval: 300 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle7 = Config{
		Frames:   []string{"⦾", "⦿"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle8 = Config{
		Frames:   []string{"◍", "◌"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle9 = Config{
		Frames:   []string{"◉", "◎"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Triangle = Config{
		Frames:   []string{"◢", "◣", "◤", "◥"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	Weather = Config{
		Frames: []string{
			"☀️ ",
			"☀️ ",
			"☀️ ",
			"🌤 ",
			"⛅️ ",
			"🌥 ",
			"☁️ ",
			"🌧 ",
			"🌨 ",
			"🌧 ",
			"🌨 ",
			"🌧 ",
			"🌨 ",
			"⛈ ",
			"🌨 ",
			"🌧 ",
			"🌨 ",
			"☁️ ",
			"🌥 ",
			"⛅️ ",
			"🌤 ",
			"☀️ ",
			"☀️ ",
		},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
)
