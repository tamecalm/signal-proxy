package ui

import (
	"fmt"
	"time"
	"github.com/fatih/color"
)

var (
	clrDim     = color.New(color.FgHiBlack)
	clrSubtle  = color.New(color.FgWhite)
	clrAccent  = color.New(color.FgCyan, color.Bold)
	clrSuccess = color.New(color.FgGreen)
	clrError   = color.New(color.FgRed)
)

func PrintBanner() {
	fmt.Println()
	badge := color.New(color.BgWhite, color.FgBlack).Sprint(" SIGNAL ")
	fmt.Printf("%s %s\n", badge, clrSubtle.Sprint("Trusted Proxy Service"))
	clrDim.Println("──────────────────────────────────────────────────────────────────")
}

func LogStatus(category, message string) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	icon := clrDim.Sprint("•")
	if category == "success" { icon = clrSuccess.Sprint("✔") }
	if category == "error" { icon = clrError.Sprint("✖") }
	fmt.Printf("%s %s %s\n", ts, icon, message)
}

func LogRelay(sni, clientIP string, up, down int64) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	
	// FIXED ALIGNMENT LOGIC:
	// %-28s: SNI column
	// %-18s: IP column
	// %-8s: Data columns
	fmt.Printf("%s %s %-28s %-18s %s %-8s %s %-8s\n", 
		ts, 
		clrSuccess.Sprint("→"), 
		clrAccent.Sprint(sni), 
		clrDim.Sprint(clientIP),
		clrDim.Sprint("↑"), formatBytes(up),
		clrDim.Sprint("↓"), formatBytes(down))
}

func formatBytes(b int64) string {
	if b < 1024 { return fmt.Sprintf("%dB", b) }
	if b < 1024*1024 { return fmt.Sprintf("%.1fKB", float64(b)/1024) }
	return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
}
