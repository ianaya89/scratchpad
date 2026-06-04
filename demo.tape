# vhs tape for pad — render with: vhs demo.tape
Output demo.gif

Set Shell zsh
Set FontSize 18
Set Width 1100
Set Height 680
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms
Set Padding 20

# use a throwaway data dir so the demo is clean
Hide
Type "export PAD_DIR=$(mktemp -d) && clear"
Enter
Show

Type "pad demo"
Enter
Sleep 1.2s

# write a markdown note
Type "# Shopping"
Enter
Enter
Type "- **milk**"
Enter
Type "- eggs"
Enter
Type "- coffee"
Sleep 800ms

# markdown preview
Ctrl+o
Sleep 2s
Escape
Sleep 400ms

# new tab + rename
Ctrl+t
Sleep 500ms
Type "ship release"
Enter
Type "deploy on friday"
Sleep 600ms
Ctrl+r
Sleep 400ms
Type "todo"
Enter
Sleep 800ms

# search across tabs, jump to a match
Ctrl+f
Sleep 500ms
Type "milk"
Sleep 1s
Enter
Sleep 1.2s

# help overlay
Ctrl+g
Sleep 2s
Escape
Sleep 500ms

Ctrl+q
Sleep 800ms
