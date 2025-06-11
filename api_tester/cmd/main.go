package main

import (
	"context"
	"os"
	"strconv"

	"github.com/bilalthdeveloper/kadrion/internal/flv"
	"github.com/bilalthdeveloper/kadrion/internal/hls"
	"github.com/bilalthdeveloper/kadrion/internal/sse"
	"github.com/bilalthdeveloper/kadrion/internal/ws"
	"github.com/bilalthdeveloper/kadrion/utils"
)

func main() {

	if len(os.Args) < 6 {
		utils.LogMessage("Missing required arguments: addr(string), count(int), timeout seconds(int), and connection type.", utils.Fatal_Error_Code)
		return
	}
	ctx := context.Background()

	addr := os.Args[1]
	Type := os.Args[5]
	PumpCount, err := strconv.ParseInt(os.Args[4], 10, 64)
	if err != nil {
		utils.LogMessage(err.Error(), 1)
	}
	initialCount, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		utils.LogMessage(err.Error(), 1)
	}
	duration, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		utils.LogMessage(err.Error(), 1)
	}

	switch Type {
	case "ws":
		ws.RunWebsocketTest(ctx, addr, initialCount, PumpCount, duration)
	case "sse":
		sse.RunSseTest(ctx, addr, initialCount, PumpCount, duration)
	case "hls":
		hls.RunHlsTest(ctx, addr, initialCount, PumpCount, duration)
	case "flv":
		flv.RunFlvTest(ctx, addr, initialCount, PumpCount, duration)
	}

}
