package commands

import (
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/attribute"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func registerAttribute(s *server.Server) {
	s.Commander.Register(
		Literal("attribute").Requires(2).Connect(
			Argument("target", parsers.Entity.Single(true)).Connect(
				Argument("attribute", parsers.Resource(attribute.Registry)).Connect(
					Literal("get").
						Executes(attrGet(s, false)).
						Connect(
							Argument("scale", parsers.Double).
								Executes(attrGet(s, true)),
						),
					Literal("base").Connect(
						Literal("get").
							Executes(attrBaseGet(s, false)).
							Connect(
								Argument("scale", parsers.Double).
									Executes(attrBaseGet(s, true)),
							),
						Literal("set").Connect(
							Argument("value", parsers.Double).
								Executes(attrBaseSet(s)),
						),
						Literal("reset").
							Executes(attrBaseReset(s)),
					),
					Literal("modifier").Connect(
						Literal("add").Connect(
							Argument("id", parsers.ResourceLocation).Connect(
								Argument("value", parsers.Double).Connect(
									Literal("add_value").
										Executes(attrModifierAdd(s, attribute.AddValue)),
									Literal("add_multiplied_base").
										Executes(attrModifierAdd(s, attribute.AddMultipliedBase)),
									Literal("add_multiplied_total").
										Executes(attrModifierAdd(s, attribute.AddMultipliedTotal)),
								),
							),
						),
						Literal("remove").Connect(
							Argument("id", parsers.ResourceLocation).
								Executes(attrModifierRemove(s)),
						),
						Literal("value").Connect(
							Literal("get").Connect(
								Argument("id", parsers.ResourceLocation).
									Executes(attrModifierValueGet(s, false)).
									Connect(
										Argument("scale", parsers.Double).
											Executes(attrModifierValueGet(s, true)),
									),
							),
						),
					),
				),
			),
		),
	)
}

func attributeSet(s *server.Server, cc *CommandContext) (entity.Entity, *attribute.Set, bool) {
	target, ok := resolveSingleEntity(s, cc, "target")
	if !ok {
		return nil, nil, false
	}
	living, ok := target.(interface {
		LivingBase() *entity.LivingEntity
	})
	if !ok {
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeFailedEntity,
			entityDisplayName(target)).SetColor(tc.ColorRed))
		return nil, nil, false
	}
	return target, living.LivingBase().Attributes, true
}

// attrScale reads the optional "scale" arg, defaulting to 1.
func attrScale(cc *CommandContext, present bool) float64 {
	if present {
		return GetArgument[float64](cc.Args, "scale")
	}
	return 1
}

func attributeName(id attribute.ID) tc.Component {
	return tc.Text("minecraft:" + id.String())
}

func modifierName(id mc.Identifier) tc.Component {
	return tc.Text(string(id))
}

func formatAttrValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func attrGet(s *server.Server, withScale bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		value := set.Value(id)
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeValueGetSuccess,
			attributeName(id), entityDisplayName(target), tc.Text(formatAttrValue(value))))
		return &CommandResult{Success: 1, Result: int(value * attrScale(cc, withScale))}, nil
	}
}

func attrBaseGet(s *server.Server, withScale bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		base := set.BaseValue(id)
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeBaseValueGetSuccess,
			attributeName(id), entityDisplayName(target), tc.Text(formatAttrValue(base))))
		return &CommandResult{Success: 1, Result: int(base * attrScale(cc, withScale))}, nil
	}
}

func attrBaseSet(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		value := GetArgument[float64](cc.Args, "value")
		set.GetOrCreate(id).SetBase(value)
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeBaseValueSetSuccess,
			attributeName(id), entityDisplayName(target), tc.Text(formatAttrValue(value))))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func attrBaseReset(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		inst := set.GetOrCreate(id)
		inst.Reset()
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeBaseValueResetSuccess,
			attributeName(id), entityDisplayName(target), tc.Text(formatAttrValue(inst.Base()))))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func attrModifierAdd(s *server.Server, op attribute.Operation) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		modID := GetArgument[mc.Identifier](cc.Args, "id")
		value := GetArgument[float64](cc.Args, "value")
		inst := set.GetOrCreate(id)
		if !inst.AddModifier(attribute.Modifier{ID: modID, Amount: value, Operation: op}) {
			cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeFailedModifierAlreadyPresent,
				modifierName(modID), attributeName(id), entityDisplayName(target)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeModifierAddSuccess,
			modifierName(modID), attributeName(id), entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func attrModifierRemove(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		modID := GetArgument[mc.Identifier](cc.Args, "id")
		inst := set.Get(id)
		if inst == nil || !inst.RemoveModifier(modID) {
			cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeFailedNoModifier,
				attributeName(id), entityDisplayName(target), modifierName(modID)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeModifierRemoveSuccess,
			modifierName(modID), attributeName(id), entityDisplayName(target)))
		return &CommandResult{Success: 1, Result: 1}, nil
	}
}

func attrModifierValueGet(s *server.Server, withScale bool) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target, set, ok := attributeSet(s, cc)
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		id := GetArgument[attribute.ID](cc.Args, "attribute")
		modID := GetArgument[mc.Identifier](cc.Args, "id")
		var modifier attribute.Modifier
		if inst := set.Get(id); inst != nil {
			modifier, ok = inst.Modifier(modID)
		} else {
			ok = false
		}
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeFailedNoModifier,
				attributeName(id), entityDisplayName(target), modifierName(modID)).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		cc.SendMessage(tc.Translatable(mcdata.CommandsAttributeModifierValueGetSuccess,
			modifierName(modID), attributeName(id), entityDisplayName(target),
			tc.Text(formatAttrValue(modifier.Amount))))
		return &CommandResult{Success: 1, Result: int(modifier.Amount * attrScale(cc, withScale))}, nil
	}
}
