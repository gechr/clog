package clog

import "time"

// Predefined spinner frame sets adapted from https://github.com/sindresorhus/cli-spinners
// Pass any of these to [AnimationBuilder.Style] to change the animation style.
var (
	SpinnerAesthetic = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerArc = SpinnerStyle{
		Frames: []string{"◜", "◠", "◝", "◞", "◡", "◟"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerArrow2 = SpinnerStyle{
		Frames: []string{"⬆️ ", "↗️ ", "➡️ ", "↘️ ", "⬇️ ", "↙️ ", "⬅️ ", "↖️ "},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerArrow3 = SpinnerStyle{
		Frames: []string{"▹▹▹▹▹", "▸▹▹▹▹", "▹▸▹▹▹", "▹▹▸▹▹", "▹▹▹▸▹", "▹▹▹▹▸"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBalloon = SpinnerStyle{
		Frames: []string{" ", ".", "o", "O", "@", "*", " "},
		FPS:    140 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBalloon2 = SpinnerStyle{
		Frames: []string{".", "o", "O", "°", "O", "o", "."},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBetaWave = SpinnerStyle{
		Frames: []string{
			"ρββββββ",
			"βρβββββ",
			"ββρββββ",
			"βββρβββ",
			"ββββρββ",
			"βββββρβ",
			"ββββββρ",
		},
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBinary = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBluePulse = SpinnerStyle{
		Frames: []string{"🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBouncingBall = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBoxBounce = SpinnerStyle{
		Frames: []string{"▖", "▘", "▝", "▗"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerBoxBounce2 = SpinnerStyle{
		Frames: []string{"▌", "▀", "▐", "▄"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerChristmas = SpinnerStyle{
		Frames: []string{"🌲", "🎄"},
		FPS:    400 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerCircle = SpinnerStyle{
		Frames: []string{"◡", "⊙", "◠"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerCircleHalves = SpinnerStyle{
		Frames: []string{"◐", "◓", "◑", "◒"},
		FPS:    50 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerCircleQuarters = SpinnerStyle{
		Frames: []string{"◴", "◷", "◶", "◵"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDot = SpinnerStyle{
		Frames: []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots = SpinnerStyle{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots11 = SpinnerStyle{
		Frames: []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots12 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots13 = SpinnerStyle{
		Frames: []string{"⣼", "⣹", "⢻", "⠿", "⡟", "⣏", "⣧", "⣶"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots14 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots3 = SpinnerStyle{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots4 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots5 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots6 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots7 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots8 = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots8Bit = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDots9 = SpinnerStyle{
		Frames: []string{"⢹", "⢺", "⢼", "⣸", "⣇", "⡧", "⡗", "⡏"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDotsCircle = SpinnerStyle{
		Frames: []string{"⢎ ", "⠎⠁", "⠊⠑", "⠈⠱", " ⡱", "⢀⡰", "⢄⡠", "⢆⡀"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDqpb = SpinnerStyle{
		Frames: []string{"d", "q", "p", "b"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerDwarfFortress = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerEllipsis = SpinnerStyle{
		Frames: []string{"", ".", "..", "..."},
		FPS:    333 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerFingerDance = SpinnerStyle{
		Frames: []string{"🤘 ", "🤟 ", "🖖 ", "✋ ", "🤚 ", "👆 "},
		FPS:    160 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerFish = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerFistBump = SpinnerStyle{
		Frames: []string{
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"🤜\u3000\u3000\u3000\u3000🤛 ",
			"\u3000🤜\u3000\u3000🤛\u3000 ",
			"\u3000\u3000🤜🤛\u3000\u3000 ",
			"\u3000🤜✨🤛\u3000\u3000 ",
			"🤜\u3000✨\u3000🤛\u3000 ",
		},
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerFlip = SpinnerStyle{
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
		FPS: 70 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerGlobe = SpinnerStyle{
		Frames: []string{"🌍", "🌎", "🌏"},
		FPS:    250 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerGrenade = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerGrowHorizontal = SpinnerStyle{
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
		FPS: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerGrowVertical = SpinnerStyle{
		Frames: []string{"▁", "▃", "▄", "▅", "▆", "▇", "▆", "▅", "▄", "▃"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerHamburger = SpinnerStyle{
		Frames: []string{"☱", "☲", "☴", "☲"},
		FPS:    333 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerJump = SpinnerStyle{
		Frames: []string{"⢄", "⢂", "⢁", "⡁", "⡈", "⡐", "⡠"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerLayer = SpinnerStyle{
		Frames: []string{"-", "=", "≡"},
		FPS:    150 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerLine = SpinnerStyle{
		Frames: []string{"|", "/", "-", "\\"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerLine2 = SpinnerStyle{
		Frames: []string{"⠂", "-", "–", "-", "–", "-"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMaterial = SpinnerStyle{
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
		FPS: 17 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMeter = SpinnerStyle{
		Frames: []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▰▰▱", "▰▱▱", "▱▱▱"},
		FPS:    143 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMindblown = SpinnerStyle{
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
		FPS: 160 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMiniDot = SpinnerStyle{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    83 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMonkey = SpinnerStyle{
		Frames: []string{"🙈", "🙉", "🙊"},
		FPS:    333 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerMoon = SpinnerStyle{
		Frames: []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
		FPS:    125 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerNoise = SpinnerStyle{
		Frames: []string{"▓", "▒", "░"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerOrangeBluePulse = SpinnerStyle{
		Frames: []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 ", "🔹 ", "🔷 ", "🔵 ", "🔵 ", "🔷 "},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerOrangePulse = SpinnerStyle{
		Frames: []string{"🔸 ", "🔶 ", "🟠 ", "🟠 ", "🔶 "},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerPipe = SpinnerStyle{
		Frames: []string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerPoint = SpinnerStyle{
		Frames: []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"},
		FPS:    125 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerPoints = SpinnerStyle{
		Frames: []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"},
		FPS:    143 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerPong = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerPulse = SpinnerStyle{
		Frames: []string{"█", "▓", "▒", "░"},
		FPS:    125 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerRollingLine = SpinnerStyle{
		Frames: []string{"/  ", " - ", " \\ ", "  |", "  |", " \\ ", " - ", "/  "},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerRunner = SpinnerStyle{
		Frames: []string{"🚶 ", "🏃 "},
		FPS:    140 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSand = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerShark = SpinnerStyle{
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
		FPS: 120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSimpleDots = SpinnerStyle{
		Frames: []string{".  ", ".. ", "...", "   "},
		FPS:    400 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSimpleDotsScrolling = SpinnerStyle{
		Frames: []string{".  ", ".. ", "...", " ..", "  .", "   "},
		FPS:    200 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSmiley = SpinnerStyle{
		Frames: []string{"😄 ", "😝 "},
		FPS:    200 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSoccerHeader = SpinnerStyle{
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
		FPS: 80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSpeaker = SpinnerStyle{
		Frames: []string{"🔈 ", "🔉 ", "🔊 ", "🔉 "},
		FPS:    160 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSquareCorners = SpinnerStyle{
		Frames: []string{"◰", "◳", "◲", "◱"},
		FPS:    180 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerSquish = SpinnerStyle{
		Frames: []string{"╫", "╪"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerStar2 = SpinnerStyle{
		Frames: []string{"+", "x", "*"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerTimeTravel = SpinnerStyle{
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
		FPS: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle = SpinnerStyle{
		Frames: []string{"⊶", "⊷"},
		FPS:    250 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle10 = SpinnerStyle{
		Frames: []string{"㊂", "㊀", "㊁"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle11 = SpinnerStyle{
		Frames: []string{"⧇", "⧆"},
		FPS:    50 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle12 = SpinnerStyle{
		Frames: []string{"☗", "☖"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle13 = SpinnerStyle{
		Frames: []string{"=", "*", "-"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle2 = SpinnerStyle{
		Frames: []string{"▫", "▪"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle3 = SpinnerStyle{
		Frames: []string{"□", "■"},
		FPS:    120 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle4 = SpinnerStyle{
		Frames: []string{"■", "□", "▪", "▫"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle5 = SpinnerStyle{
		Frames: []string{"▮", "▯"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle6 = SpinnerStyle{
		Frames: []string{"ဝ", "၀"},
		FPS:    300 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle7 = SpinnerStyle{
		Frames: []string{"⦾", "⦿"},
		FPS:    80 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle8 = SpinnerStyle{
		Frames: []string{"◍", "◌"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerToggle9 = SpinnerStyle{
		Frames: []string{"◉", "◎"},
		FPS:    100 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerTriangle = SpinnerStyle{
		Frames: []string{"◢", "◣", "◤", "◥"},
		FPS:    50 * time.Millisecond, //nolint:mnd // frame rate
	}
	SpinnerWeather = SpinnerStyle{
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
		FPS: 100 * time.Millisecond, //nolint:mnd // frame rate
	}
)
