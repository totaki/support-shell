package shell

func traceHeader(color bool) string {
 if color { return "  \x1b[38;5;245m┌ Tools\x1b[0m" }
 return "  ┌ Tools"
}
func traceFooter(color bool) string {
 if color { return "  \x1b[38;5;245m└──────────────\x1b[0m\n  \x1b[38;5;245mAnswer\x1b[0m" }
 return "  └──────────────\n  Answer"
}
