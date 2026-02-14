Output ./assets/demo.gif

Set Shell bash
Set FontSize 22
Set Framerate 60
Set Width 1396
Set Height 266
Set Padding 15
Set Margin 10
Set MarginFill "#674EFF"
Set BorderRadius 10

Env PS1 "\[\e[38;5;208m\]$\[\e[0m\] "
Env DEMO "1"
Hide
Type "go run ./examples"
Show
Sleep 100ms
Enter
Sleep 15s
