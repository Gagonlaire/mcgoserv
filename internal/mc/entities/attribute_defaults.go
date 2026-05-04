package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/attribute"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

var attributeDefaults = map[mcdata.EntityType][]attribute.Default{
	mcdata.EntityZombie: {
		{ID: attribute.Armor, Base: 2},
		{ID: attribute.AttackDamage, Base: 3},
		{ID: attribute.FollowRange, Base: 35},
		{ID: attribute.MovementSpeed, Base: 0.23},
		{ID: attribute.SpawnReinforcements, Base: attribute.SpawnReinforcements.Default()},
	},
}

var (
	rollHorseJumpStrength    = attribute.Roll{ID: attribute.JumpStrength, Min: 0.4, Max: 1.0}
	rollEquineMaxHealth      = attribute.Roll{ID: attribute.MaxHealth, Min: 15, Max: 30}
	rollEquineSpeed          = attribute.Roll{ID: attribute.MovementSpeed, Min: 0.1125, Max: 0.3375}
	rollZombieHorseSpeed     = attribute.Roll{ID: attribute.MovementSpeed, Min: 0.2135, Max: 0.2846}
	rollZombieReinforcements = attribute.Roll{ID: attribute.SpawnReinforcements, Min: 0, Max: 0.1}
)

var rollableAttributes = map[mcdata.EntityType][]attribute.Roll{
	mcdata.EntityHorse:          {rollHorseJumpStrength, rollEquineMaxHealth, rollEquineSpeed},
	mcdata.EntitySkeletonHorse:  {rollHorseJumpStrength},
	mcdata.EntityZombieHorse:    {rollHorseJumpStrength, rollZombieHorseSpeed},
	mcdata.EntityDonkey:         {rollEquineMaxHealth, rollEquineSpeed},
	mcdata.EntityMule:           {rollEquineMaxHealth, rollEquineSpeed},
	mcdata.EntityLlama:          {rollEquineMaxHealth},
	mcdata.EntityTraderLlama:    {rollEquineMaxHealth},
	mcdata.EntityZombie:         {rollZombieReinforcements},
	mcdata.EntityHusk:           {rollZombieReinforcements},
	mcdata.EntityDrowned:        {rollZombieReinforcements},
	mcdata.EntityZombieVillager: {rollZombieReinforcements},
}
