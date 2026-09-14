# prpr's README demo, recorded with VHS: https://github.com/charmbracelet/vhs
#
#   make demo
#
# The recording runs against --demo, which serves a fixed fixture list instead
# of GitHub. That keeps whoever re-records this from publishing the contents of
# their own account, and makes the result the same every time.
#
# The fixtures merge one pull request about eleven seconds in, so the timings
# below are load-bearing: the walkthrough has to still be running when the
# refresh that drops it lands, or the farewell banner never appears on camera.
#
# Recorded with VHS 0.11.0. VHS 0.12.0 exits successfully without writing the
# GIF at all on macOS with ffmpeg 9, so check that docs/demo.gif was actually
# rewritten rather than trusting the "Creating ..." line.

Output docs/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1080
Set Height 680
Set Padding 24
Set Margin 12
Set MarginFill "#1a1a2e"
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set Framerate 24
Set TypingSpeed 55ms

Type "prpr --demo --interval 5s"
Sleep 600ms
Enter

# Owner discovery, then the first fetch.
Sleep 3s

# Walk the list.
Down Sleep 700ms
Down Sleep 700ms
Down Sleep 900ms

# The detail pane, side by side at this width.
Type "d"
Sleep 2s
Ctrl+D
Sleep 1200ms
Type "d"
Sleep 600ms

# Filter down to one repository, then clear it.
Type "/"
Sleep 400ms
Type "worker"
Sleep 1500ms
Enter
Sleep 1200ms
Escape
Sleep 800ms

# Tabs.
Tab Sleep 1200ms
Tab Sleep 1200ms
Tab Sleep 1200ms
Tab Sleep 1s

# Wait for the refresh that finds #127 merged, and let the banner be read.
Sleep 6s

Type "q"
Sleep 1500ms
