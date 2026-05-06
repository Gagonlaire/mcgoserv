package entity

import "github.com/Gagonlaire/mcgoserv/internal/mc/attribute"

var attributeDefaults = map[ID][]attribute.Default{
	ZombieID: {
		{ID: attribute.ArmorID, Base: 2},
		{ID: attribute.AttackDamageID, Base: 3},
		{ID: attribute.FollowRangeID, Base: 35},
		{ID: attribute.MovementSpeedID, Base: 0.23},
		{ID: attribute.SpawnReinforcementsID, Base: attribute.SpawnReinforcementsID.Default()},
	},
}

var (
	rollHorseJumpStrength    = attribute.Roll{ID: attribute.JumpStrengthID, Min: 0.4, Max: 1.0}
	rollEquineMaxHealth      = attribute.Roll{ID: attribute.MaxHealthID, Min: 15, Max: 30}
	rollEquineSpeed          = attribute.Roll{ID: attribute.MovementSpeedID, Min: 0.1125, Max: 0.3375}
	rollZombieHorseSpeed     = attribute.Roll{ID: attribute.MovementSpeedID, Min: 0.2135, Max: 0.2846}
	rollZombieReinforcements = attribute.Roll{ID: attribute.SpawnReinforcementsID, Min: 0, Max: 0.1}
)

var rollableAttributes = map[ID][]attribute.Roll{
	HorseID:          {rollHorseJumpStrength, rollEquineMaxHealth, rollEquineSpeed},
	SkeletonHorseID:  {rollHorseJumpStrength},
	ZombieHorseID:    {rollHorseJumpStrength, rollZombieHorseSpeed},
	DonkeyID:         {rollEquineMaxHealth, rollEquineSpeed},
	MuleID:           {rollEquineMaxHealth, rollEquineSpeed},
	LlamaID:          {rollEquineMaxHealth},
	TraderLlamaID:    {rollEquineMaxHealth},
	ZombieID:         {rollZombieReinforcements},
	HuskID:           {rollZombieReinforcements},
	DrownedID:        {rollZombieReinforcements},
	ZombieVillagerID: {rollZombieReinforcements},
}
