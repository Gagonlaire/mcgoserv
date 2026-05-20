package commands

import (
	"errors"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/Tnze/go-mc/nbt"
	"github.com/google/uuid"
)

func registerData(s *server.Server) {
	s.Commander.Register(
		Literal("data").Connect(
			Literal("get").Connect(
				Literal("entity").Connect(
					Argument("target", parsers.Entity.Single(true)).
						Executes(dataGetEntity(s)).
						Connect(
							Argument("path", parsers.NbtPath).
								Executes(dataGetEntityPath(s)).
								Connect(
									Argument("scale", parsers.Double).
										Executes(dataGetEntityPathScale(s)),
								),
						),
				),
			),
			Literal("merge").Connect(
				Literal("entity").Connect(
					Argument("target", parsers.Entity.Single(true)).Connect(
						Argument("nbt", parsers.NbtCompoundTag).
							Executes(dataMergeEntity(s)),
					),
				),
			),
			Literal("modify").Connect(
				Literal("entity").Connect(
					Argument("target", parsers.Entity.Single(true)).Connect(
						Argument("path", parsers.NbtPath).Connect(
							Literal("set").Connect(
								Literal("value").Connect(
									Argument("nbt", parsers.NbtTag).
										Executes(dataModifyOp(s, opSet)),
								),
								Literal("from").Connect(
									Literal("entity").Connect(
										Argument("source", parsers.Entity.Single(true)).
											Executes(dataModifyFrom(s, opSet, false)).
											Connect(
												Argument("sourcePath", parsers.NbtPath).
													Executes(dataModifyFrom(s, opSet, true)),
											),
									),
								),
								// todo: extend to append/prepend/insert/merge
								Literal("string").Connect(
									Literal("entity").Connect(
										Argument("source", parsers.Entity.Single(true)).
											Executes(dataModifyString(s, opSet, false, false, false)).
											Connect(
												Argument("sourcePath", parsers.NbtPath).
													Executes(dataModifyString(s, opSet, true, false, false)).
													Connect(
														Argument("start", parsers.Int).
															Executes(dataModifyString(s, opSet, true, true, false)).
															Connect(
																Argument("end", parsers.Int).
																	Executes(dataModifyString(s, opSet, true, true, true)),
															),
													),
											),
									),
								),
							),
							Literal("append").Connect(
								Literal("value").Connect(
									Argument("nbt", parsers.NbtTag).
										Executes(dataModifyOp(s, opAppend)),
								),
								Literal("from").Connect(
									Literal("entity").Connect(
										Argument("source", parsers.Entity.Single(true)).
											Executes(dataModifyFrom(s, opAppend, false)).
											Connect(
												Argument("sourcePath", parsers.NbtPath).
													Executes(dataModifyFrom(s, opAppend, true)),
											),
									),
								),
							),
							Literal("prepend").Connect(
								Literal("value").Connect(
									Argument("nbt", parsers.NbtTag).
										Executes(dataModifyOp(s, opPrepend)),
								),
								Literal("from").Connect(
									Literal("entity").Connect(
										Argument("source", parsers.Entity.Single(true)).
											Executes(dataModifyFrom(s, opPrepend, false)).
											Connect(
												Argument("sourcePath", parsers.NbtPath).
													Executes(dataModifyFrom(s, opPrepend, true)),
											),
									),
								),
							),
							Literal("insert").Connect(
								Argument("index", parsers.Int).Connect(
									Literal("value").Connect(
										Argument("nbt", parsers.NbtTag).
											Executes(dataModifyInsert(s, false)),
									),
									Literal("from").Connect(
										Literal("entity").Connect(
											Argument("source", parsers.Entity.Single(true)).
												Executes(dataModifyInsertFrom(s, false)).
												Connect(
													Argument("sourcePath", parsers.NbtPath).
														Executes(dataModifyInsertFrom(s, true)),
												),
										),
									),
								),
							),
							Literal("merge").Connect(
								Literal("value").Connect(
									Argument("nbt", parsers.NbtCompoundTag).
										Executes(dataModifyMergeValue(s)),
								),
								Literal("from").Connect(
									Literal("entity").Connect(
										Argument("source", parsers.Entity.Single(true)).
											Executes(dataModifyMergeFrom(s, false)).
											Connect(
												Argument("sourcePath", parsers.NbtPath).
													Executes(dataModifyMergeFrom(s, true)),
											),
									),
								),
							),
						),
					),
				),
			),
			Literal("remove").Connect(
				Literal("entity").Connect(
					Argument("target", parsers.Entity.Single(true)).Connect(
						Argument("path", parsers.NbtPath).
							Executes(dataRemoveEntity(s)),
					),
				),
			),
		),
	)
}

// modifyOp is the kind of op used by /data modify ... (excluding insert,
// which carries an index).
type modifyOp int

const (
	opSet modifyOp = iota
	opAppend
	opPrepend
)

// resolveSingleEntity reads target arg `name` and resolves to one entity.
// On miss, it sends the standard "not found" message and returns false.
func resolveSingleEntity(s *server.Server, cc *CommandContext, name string) (entity.Entity, bool) {
	player := cc.Source.Entity.(*entity.Player)
	targets := cc.Args.GetEntityTarget(name)
	resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
	if len(resolved) == 0 {
		cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
		return nil, false
	}
	return resolved[0], true
}

// asWriteTarget asserts e implements NbtTarget AND is not a player (vanilla
// forbids player NBT writes). Sends the appropriate error and returns false
// if either gate fails.
func asWriteTarget(cc *CommandContext, e entity.Entity) (nbtpath.NbtTarget, bool) {
	if _, isPlayer := e.(*entity.Player); isPlayer {
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
		return nil, false
	}
	tgt, ok := e.(nbtpath.NbtTarget)
	if !ok {
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
		return nil, false
	}
	return tgt, true
}

// nbtErrKey maps an nbtpath sentinel to the appropriate vanilla
// translation key for /data error messages.
func nbtErrKey(err error) mcdata.TranslationKey {
	switch {
	case errors.Is(err, nbtpath.ErrPathNotFound):
		return mcdata.CommandsDataGetUnknown
	case errors.Is(err, nbtpath.ErrNotAList):
		return mcdata.CommandsDataModifyExpectedList
	case errors.Is(err, nbtpath.ErrNotACompound), errors.Is(err, nbtpath.ErrExpectedCompound):
		return mcdata.CommandsDataModifyExpectedObject
	case errors.Is(err, nbtpath.ErrIndexOOB):
		return mcdata.CommandsDataModifyInvalidIndex
	default:
		return mcdata.CommandsDataMergeFailed
	}
}

// floorInt implements the vanilla rounding formula: n < int(n) ? int(n)-1 : int(n).
func floorInt(f float64) int {
	n := int(f)
	if float64(n) > f {
		return n - 1
	}
	return n
}

// nbtAsFloat64 returns the float64 representation of a numeric NBT value.
func nbtAsFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	}
	return 0, false
}

func nbtResultValue(v any) int {
	switch val := v.(type) {
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return floorInt(float64(val))
	case float32:
		return floorInt(float64(val))
	case float64:
		return floorInt(val)
	case string:
		return len(val)
	case []any:
		return len(val)
	case map[string]any:
		return len(val)
	case []byte:
		return len(val)
	case []int32:
		return len(val)
	case []int64:
		return len(val)
	default:
		return 1
	}
}

func dataGetEntity(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		src, ok := target.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		root, err := src.NbtRoot()
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		comp := nbtpath.FormatSNBTComponent(root)
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityQuery, entityDisplayName(target), comp))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func dataGetEntityPath(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		src, ok := target.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		root, err := src.NbtRoot()
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		anchors, err := nbtpath.Resolve(root, *path)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if len(anchors) > 1 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetMultiple).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		comp := nbtpath.FormatSNBTComponent(anchors[0].Value())
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityGet,
			tc.Text(path.Raw), entityDisplayName(target), comp))
		return &CommandResult{Success: 1, Result: nbtResultValue(anchors[0].Value())}, nil
	}
}

func dataGetEntityPathScale(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		src, ok := target.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		root, err := src.NbtRoot()
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		anchors, err := nbtpath.Resolve(root, *path)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if len(anchors) > 1 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetMultiple).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		f, ok := nbtAsFloat64(anchors[0].Value())
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		scale := GetArgument[float64](cc.Args, "scale")
		result := floorInt(f * scale)
		comp := nbtpath.FormatSNBTComponent(anchors[0].Value())
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityGet,
			tc.Text(path.Raw), entityDisplayName(target), comp))
		return &CommandResult{Success: 1, Result: result}, nil
	}
}

func dataMergeEntity(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		compound := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
		v, err := nbtpath.SNBTToValue(compound)
		if err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		src, ok := v.(map[string]any)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if _, err := tgt.NbtMerge(src); err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func dataModifyOp(s *server.Server, op modifyOp) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		raw := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
		v, err := nbtpath.SNBTToValue(raw)
		if err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataModifyExpectedValue).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := applyOp(tgt, op, *path, v)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyFrom(s *server.Server, op modifyOp, withSourcePath bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		source, ok := resolveSingleEntity(s, cc, "source")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		srcSource, ok := source.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		var srcPath nbtpath.Path
		if withSourcePath {
			sp := GetArgument[*nbtpath.Path](cc.Args, "sourcePath")
			srcPath = *sp
		}
		fv := nbtpath.FromValueSource{Src: srcSource, Path: srcPath}
		values, err := fv.Resolve()
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if len(values) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetUnknown).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		// Vanilla applies multi-source values one by one — for the
		// single-target ops here we use the first match.
		n, err := applyOp(tgt, op, *path, values[0])
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyInsert(s *server.Server, _ bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		idx := GetArgument[int32](cc.Args, "index")
		raw := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
		v, err := nbtpath.SNBTToValue(raw)
		if err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataModifyExpectedValue).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := tgt.NbtInsert(*path, int(idx), v)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyInsertFrom(s *server.Server, withSourcePath bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		source, ok := resolveSingleEntity(s, cc, "source")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		srcSource, ok := source.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		idx := GetArgument[int32](cc.Args, "index")
		var srcPath nbtpath.Path
		if withSourcePath {
			sp := GetArgument[*nbtpath.Path](cc.Args, "sourcePath")
			srcPath = *sp
		}
		fv := nbtpath.FromValueSource{Src: srcSource, Path: srcPath}
		values, err := fv.Resolve()
		if err != nil || len(values) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetUnknown).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := tgt.NbtInsert(*path, int(idx), values[0])
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyMergeValue(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		raw := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
		v, err := nbtpath.SNBTToValue(raw)
		if err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		compound, ok := v.(map[string]any)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataModifyExpectedObject).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := tgt.NbtMergeAt(*path, compound)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyMergeFrom(s *server.Server, withSourcePath bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		source, ok := resolveSingleEntity(s, cc, "source")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		srcSource, ok := source.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		var srcPath nbtpath.Path
		if withSourcePath {
			sp := GetArgument[*nbtpath.Path](cc.Args, "sourcePath")
			srcPath = *sp
		}
		fv := nbtpath.FromValueSource{Src: srcSource, Path: srcPath}
		values, err := fv.Resolve()
		if err != nil || len(values) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetUnknown).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		compound, ok := values[0].(map[string]any)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataModifyExpectedObject).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := tgt.NbtMergeAt(*path, compound)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataModifyString(s *server.Server, op modifyOp, withSourcePath, withStart, withEnd bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		source, ok := resolveSingleEntity(s, cc, "source")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		srcSource, ok := source.(nbtpath.NbtSource)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		var srcPath nbtpath.Path
		if withSourcePath {
			sp := GetArgument[*nbtpath.Path](cc.Args, "sourcePath")
			srcPath = *sp
		}
		sv := nbtpath.StringValueSource{Src: srcSource, Path: srcPath}
		if withStart {
			start := int(GetArgument[int32](cc.Args, "start"))
			sv.Start = &start
		}
		if withEnd {
			end := int(GetArgument[int32](cc.Args, "end"))
			sv.End = &end
		}
		values, err := sv.Resolve()
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		n, err := applyOp(tgt, op, *path, values[0])
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func dataRemoveEntity(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, ok := resolveSingleEntity(s, cc, "target")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		tgt, ok := asWriteTarget(cc, target)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		path := GetArgument[*nbtpath.Path](cc.Args, "path")
		n, err := tgt.NbtRemove(*path)
		if err != nil {
			cc.SendMessage(tc.Translatable(nbtErrKey(err)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if n == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: n}, nil
	}
}

func applyOp(tgt nbtpath.NbtTarget, op modifyOp, p nbtpath.Path, v any) (int, error) {
	switch op {
	case opSet:
		return tgt.NbtSet(p, v)
	case opAppend:
		return tgt.NbtAppend(p, v)
	case opPrepend:
		return tgt.NbtPrepend(p, v)
	}
	return 0, errors.New("unknown op")
}
