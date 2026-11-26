Set Columns 37
Set Rows 11
Source examples/base.tape

Hide
Type@0 'clear && exec ./.demo-bin/spinner -styles-page "$DEMO_PAGE"'
Enter
Wait+Screen /(?m)^[A-Z][A-Za-z0-9]+ {2,}\S/
Sleep 80ms
Show
Sleep 9s
