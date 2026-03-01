package spinner

import "time"

// Predefined spinner frame sets adapted from https://github.com/sindresorhus/cli-spinners
// Pass any of these to [WithStyle] or the animation builder's Style method to change the animation.
var (
	Aesthetic = Style{
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
	Arc = Style{
		Frames:   []string{"◜", "◠", "◝", "◞", "◡", "◟"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Arrow2 = Style{
		Frames:   []string{"⬆️ ", "↗️ ", "➡️ ", "↘️ ", "⬇️ ", "↙️ ", "⬅️ ", "↖️ "},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Arrow3 = Style{
		Frames:   []string{"▹▹▹▹▹", "▸▹▹▹▹", "▹▸▹▹▹", "▹▹▸▹▹", "▹▹▹▸▹", "▹▹▹▹▸"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Balloon = Style{
		Frames:   []string{" ", ".", "o", "O", "@", "*", " "},
		Interval: 140 * time.Millisecond, //nolint:mnd // frame rate
	}
	Balloon2 = Style{
		Frames:   []string{".", "o", "O", "°", "O", "o", "."},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	BetaWave = Style{
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
	Binary = Style{
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
	BluePulse = Style{
		Frames:   []string{"🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	BouncingBall = Style{
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
	BoxBounce = Style{
		Frames:   []string{"▖", "▘", "▝", "▗"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	BoxBounce2 = Style{
		Frames:   []string{"▌", "▀", "▐", "▄"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Christmas = Style{
		Frames:   []string{"🌲", "🎄"},
		Interval: 400 * time.Millisecond, //nolint:mnd // frame rate
	}
	Circle = Style{
		Frames:   []string{"◡", "⊙", "◠"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	CircleHalves = Style{
		Frames:   []string{"◐", "◓", "◑", "◒"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	CircleQuarters = Style{
		Frames:   []string{"◴", "◷", "◶", "◵"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dot = Style{
		Frames:   []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots = Style{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots11 = Style{
		Frames:   []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots12 = Style{
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
	Dots13 = Style{
		Frames:   []string{"⣼", "⣹", "⢻", "⠿", "⡟", "⣏", "⣧", "⣶"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots14 = Style{
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
	Dots3 = Style{
		Frames:   []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dots4 = Style{
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
	Dots5 = Style{
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
	Dots6 = Style{
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
	Dots7 = Style{
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
	Dots8 = Style{
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
	Dots8Bit = Style{
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
	Dots9 = Style{
		Frames:   []string{"⢹", "⢺", "⢼", "⣸", "⣇", "⡧", "⡗", "⡏"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	DotsCircle = Style{
		Frames:   []string{"⢎ ", "⠎⠁", "⠊⠑", "⠈⠱", " ⡱", "⢀⡰", "⢄⡠", "⢆⡀"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Dqpb = Style{
		Frames:   []string{"d", "q", "p", "b"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	DwarfFortress = Style{
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
	Ellipsis = Style{
		Frames:   []string{"", ".", "..", "..."},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	FingerDance = Style{
		Frames:   []string{"🤘 ", "🤟 ", "🖖 ", "✋ ", "🤚 ", "👆 "},
		Interval: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	Fish = Style{
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
	FistBump = Style{
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
	Flip = Style{
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
	Globe = Style{
		Frames:   []string{"🌍", "🌎", "🌏"},
		Interval: 250 * time.Millisecond, //nolint:mnd // frame rate
	}
	Grenade = Style{
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
	GrowHorizontal = Style{
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
	GrowVertical = Style{
		Frames:   []string{"▁", "▃", "▄", "▅", "▆", "▇", "▆", "▅", "▄", "▃"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Hamburger = Style{
		Frames:   []string{"☱", "☲", "☴", "☲"},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	Jump = Style{
		Frames:   []string{"⢄", "⢂", "⢁", "⡁", "⡈", "⡐", "⡠"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Layer = Style{
		Frames:   []string{"-", "=", "≡"},
		Interval: 150 * time.Millisecond, //nolint:mnd // frame rate
	}
	Line = Style{
		Frames:   []string{"|", "/", "-", "\\"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Line2 = Style{
		Frames:   []string{"⠂", "-", "–", "-", "–", "-"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Material = Style{
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
	Meter = Style{
		Frames:   []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▰▰▱", "▰▱▱", "▱▱▱"},
		Interval: 143 * time.Millisecond, //nolint:mnd // frame rate
	}
	Mindblown = Style{
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
	MiniDot = Style{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 83 * time.Millisecond, //nolint:mnd // frame rate
	}
	Monkey = Style{
		Frames:   []string{"🙈", "🙉", "🙊"},
		Interval: 333 * time.Millisecond, //nolint:mnd // frame rate
	}
	Moon = Style{
		Frames:   []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	Noise = Style{
		Frames:   []string{"▓", "▒", "░"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	OrangeBluePulse = Style{
		Frames:   []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 ", "🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	OrangePulse = Style{
		Frames:   []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 "},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Pipe = Style{
		Frames:   []string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Point = Style{
		Frames:   []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	Points = Style{
		Frames:   []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"},
		Interval: 143 * time.Millisecond, //nolint:mnd // frame rate
	}
	Pong = Style{
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
	Pulse = Style{
		Frames:   []string{"█", "▓", "▒", "░"},
		Interval: 125 * time.Millisecond, //nolint:mnd // frame rate
	}
	RollingLine = Style{
		Frames:   []string{"/  ", " - ", " \\ ", "  |", "  |", " \\ ", " - ", "/  "},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Runner = Style{
		Frames:   []string{"🚶 ", "🏃 "},
		Interval: 140 * time.Millisecond, //nolint:mnd // frame rate
	}
	Sand = Style{
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
	Shark = Style{
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
	SimpleDots = Style{
		Frames:   []string{".  ", ".. ", "...", "   "},
		Interval: 400 * time.Millisecond, //nolint:mnd // frame rate
	}
	SimpleDotsScrolling = Style{
		Frames:   []string{".  ", ".. ", "...", " ..", "  .", "   "},
		Interval: 200 * time.Millisecond, //nolint:mnd // frame rate
	}
	Smiley = Style{
		Frames:   []string{"😄 ", "😝 "},
		Interval: 200 * time.Millisecond, //nolint:mnd // frame rate
	}
	SoccerHeader = Style{
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
	Speaker = Style{
		Frames:   []string{"🔈 ", "🔉 ", "🔊 ", "🔉 "},
		Interval: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	SquareCorners = Style{
		Frames:   []string{"◰", "◳", "◲", "◱"},
		Interval: 180 * time.Millisecond, //nolint:mnd // frame rate
	}
	Squish = Style{
		Frames:   []string{"╫", "╪"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Star2 = Style{
		Frames:   []string{"+", "x", "*"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	TimeTravel = Style{
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
	Toggle = Style{
		Frames:   []string{"⊶", "⊷"},
		Interval: 250 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle10 = Style{
		Frames:   []string{"㊂", "㊀", "㊁"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle11 = Style{
		Frames:   []string{"⧇", "⧆"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle12 = Style{
		Frames:   []string{"☗", "☖"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle13 = Style{
		Frames:   []string{"=", "*", "-"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle2 = Style{
		Frames:   []string{"▫", "▪"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle3 = Style{
		Frames:   []string{"□", "■"},
		Interval: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle4 = Style{
		Frames:   []string{"■", "□", "▪", "▫"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle5 = Style{
		Frames:   []string{"▮", "▯"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle6 = Style{
		Frames:   []string{"ဝ", "၀"},
		Interval: 300 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle7 = Style{
		Frames:   []string{"⦾", "⦿"},
		Interval: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle8 = Style{
		Frames:   []string{"◍", "◌"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Toggle9 = Style{
		Frames:   []string{"◉", "◎"},
		Interval: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	Triangle = Style{
		Frames:   []string{"◢", "◣", "◤", "◥"},
		Interval: 50 * time.Millisecond, //nolint:mnd // frame rate
	}
	Weather = Style{
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
