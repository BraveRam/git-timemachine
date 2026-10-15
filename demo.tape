# Charm VHS Script for git-timemachine
# To render: vhs demo.tape

Output demo.gif

Set FontSize 14
Set Width 1280
Set Height 720
Set Padding 20
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

Type "./bin/git-timemachine"
Enter
Sleep 1.5s

# Scrub forward through history
Type "l"
Sleep 300ms
Type "l"
Sleep 300ms
Type "l"
Sleep 500ms

# Jump 10 commits forward
Type "K"
Sleep 700ms

# Switch focus to diff view and scroll
Type @500ms "	"
Type "j"
Sleep 200ms
Type "j"
Sleep 400ms

# Fullscreen diff
Type "f"
Sleep 1.5s
Type "f"
Sleep 500ms

# Start timelapse playback movie
Type " "
Sleep 2.5s

# Increase playback speed to 2x and 5x
Type "]"
Sleep 1.5s
Type "]"
Sleep 2s
Type " "
Sleep 500ms

# Cycle themes (Tokyo Night, Nord, Monokai, Catppuccin)
Type "t"
Sleep 1s
Type "t"
Sleep 1s
Type "t"
Sleep 1s
Type "t"
Sleep 1s

# Search commits
Type "/"
Sleep 500ms
Type "fix"
Sleep 1s
Enter
Sleep 1s

# Open help overlay
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# Exit cleanly
Type "q"
Sleep 500ms
