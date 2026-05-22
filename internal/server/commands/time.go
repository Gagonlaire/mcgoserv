package commands

import (
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

var timeMarkers = map[string]int64{
	"day":      1000,
	"night":    13000,
	"noon":     6000,
	"midnight": 18000,
}

func registerTime(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/time set <time>", parsers.Time.Min(0)).Requires(2).Executes(timeSetValue(s))
		Build("/time set <marker: day|night|noon|midnight>").Requires(2).Executes(timeSetMarker(s))
		Build("/time add <time>", parsers.Time.Min(0)).Requires(2).Executes(timeAdd(s))
		Build("/time query <field: daytime|gametime|day>").Requires(2).Executes(timeQuery(s))
	})
}

func daylightCycleTime(s *server.Server) int64 {
	return s.World.Day*mc.TicksPerDay + s.World.DayTime
}

func setDaylightCycleTime(s *server.Server, total int64) int64 {
	if total < 0 {
		total = 0
	}
	s.World.Day = total / mc.TicksPerDay
	s.World.DayTime = total % mc.TicksPerDay

	pkt, err := packet.NewPacket(packet.PlayClientboundSetTime,
		mc.Long(s.World.Time), mc.Long(s.World.DayTime), mc.Boolean(true))
	if err == nil {
		s.World.NextTimeUpdate = s.World.Time + 20
		s.BroadcastAll(pkt)
	}
	return s.World.DayTime
}

func timeSetValue(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		dayTime := setDaylightCycleTime(s, int64(GetArgument[int32](cc.Args, "time")))
		cc.SendMessage(tc.Translatable(mcdata.CommandsTimeSet, tc.Text(strconv.FormatInt(dayTime, 10))))
		return &CommandResult{Success: 1, Result: int(dayTime)}, nil
	}
}

func timeSetMarker(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		dayTime := setDaylightCycleTime(s, timeMarkers[cc.Args.GetString("marker")])
		cc.SendMessage(tc.Translatable(mcdata.CommandsTimeSet, tc.Text(strconv.FormatInt(dayTime, 10))))
		return &CommandResult{Success: 1, Result: int(dayTime)}, nil
	}
}

func timeAdd(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		total := daylightCycleTime(s) + int64(GetArgument[int32](cc.Args, "time"))
		dayTime := setDaylightCycleTime(s, total)
		cc.SendMessage(tc.Translatable(mcdata.CommandsTimeSet, tc.Text(strconv.FormatInt(dayTime, 10))))
		return &CommandResult{Success: 1, Result: int(dayTime)}, nil
	}
}

func timeQuery(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		var value int64
		switch cc.Args.GetString("field") {
		case "daytime":
			value = s.World.DayTime
		case "gametime":
			value = s.World.Time
		case "day":
			value = s.World.Day
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsTimeQuery, tc.Text(strconv.FormatInt(value, 10))))
		return &CommandResult{Success: 1, Result: int(value)}, nil
	}
}
