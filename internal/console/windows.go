package console

import (
    "log"
    "os"
    "runtime"
    "golang.org/x/sys/windows"
)

func EnableVT() {
    if runtime.GOOS == "windows" {
        var outMode uint32
        out := windows.Handle(os.Stdout.Fd())
        if err := windows.GetConsoleMode(out, &outMode); err != nil {
            log.Printf("Warning: Failed to get console mode: %v. Colors might not work.", err)
            return
        }

        outMode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

        if err := windows.SetConsoleMode(out, outMode); err != nil {
            log.Printf("Warning: Failed to set console mode: %v. Colors might not work.", err)
        }
    }
}