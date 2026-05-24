package commands

import (
	"context"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/world"
	"github.com/google/uuid"
)

type dataEnv struct {
	exec   func(input string) (*CommandResult, error)
	zombie *entity.Zombie
	player *entity.Player
	world  *world.World
}

func newDataEnv(t *testing.T) *dataEnv {
	t.Helper()
	w := world.NewWorld()
	d := NewDispatcher()
	s := &server.Server{World: w, Commander: d}
	registerData(s)

	player := entity.NewPlayer(uuid.New(), "tester", 4, nil, &systems.Config{})
	zombie := entity.NewZombie("minecraft:overworld", [3]float64{0, 64, 0}, [2]float32{0, 0})
	if err := w.AddEntity(zombie); err != nil {
		t.Fatalf("AddEntity: %v", err)
	}

	src := &CommandSource{
		Entity:          player,
		PermissionLevel: 4,
		SendMessage:     func(any) {},
	}
	exec := func(input string) (*CommandResult, error) {
		return d.ExecuteInput(context.Background(), src, input)
	}
	return &dataEnv{exec: exec, zombie: zombie, player: player, world: w}
}

// Test 1 – data get entity @n (root dump)
func TestDataGetEntityRoot(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data get entity @n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if r.Result != 1 {
		t.Fatalf("want result=1 got %d", r.Result)
	}
}

// Test 2 – data get entity @n Health (numeric result = floor(value))
func TestDataGetEntityNumericPath(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data get entity @n Health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	want := int(env.zombie.Health) // Health=20, floor(20.0)=20
	if r.Result != want {
		t.Fatalf("want result=%d got %d", want, r.Result)
	}
}

// Test 3 – data get entity @n Health 2.0 (scale applied)
func TestDataGetEntityPathScale(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data get entity @n Health 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	want := int(env.zombie.Health * 2) // floor(20.0 * 2.0) = 40
	if r.Result != want {
		t.Fatalf("want result=%d got %d", want, r.Result)
	}
}

// Test 4 – data get entity @n Pos (list result = element count)
func TestDataGetEntityListPath(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data get entity @n Pos")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if r.Result != 3 {
		t.Fatalf("want result=3 (Pos has 3 elements) got %d", r.Result)
	}
}

// Test 5 – data get entity @n Pos 1.0 (scale on non-numeric tag → fail)
func TestDataGetEntityScaleOnNonNumeric(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data get entity @n Pos 1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 0 {
		t.Fatalf("want success=0 (scale on list tag must fail) got %d", r.Success)
	}
}

// Test 6 – data merge entity @n {Health:10f} (field updated on zombie)
func TestDataMergeEntity(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data merge entity @n {Health:10f}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if env.zombie.Health != float32(10) {
		t.Fatalf("want zombie.Health=10 got %v", env.zombie.Health)
	}
}

// Test 7 – data remove entity @n Health (tag removed from NBT map)
func TestDataRemoveEntity(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data remove entity @n Health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if r.Result < 1 {
		t.Fatalf("want result≥1 (at least one tag removed) got %d", r.Result)
	}
}

// Test 8 – data modify entity @n Health set value 5f (field updated on zombie)
func TestDataModifySetValue(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data modify entity @n Health set value 5f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if env.zombie.Health != float32(5) {
		t.Fatalf("want zombie.Health=5 got %v", env.zombie.Health)
	}
}

// Test 9 – data modify entity @n sleeping_pos append value 99 (list grows)
func TestDataModifyAppendValue(t *testing.T) {
	env := newDataEnv(t)
	if r, err := env.exec("data merge entity @n {sleeping_pos:[0,64,0]}"); err != nil || r.Success != 1 {
		t.Fatalf("setup merge failed: err=%v success=%v", err, r)
	}
	r, err := env.exec("data modify entity @n sleeping_pos append value 99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if len(env.zombie.SleepingPos) != 4 {
		t.Fatalf("want 4 elements after append got %d", len(env.zombie.SleepingPos))
	}
}

// Test 9b – data modify entity @n {} merge value {Health:5f}
func TestDataModifyMergeValue(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data modify entity @n {} merge value {Health:5f}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if env.zombie.Health != float32(5) {
		t.Fatalf("want zombie.Health=5 after merge got %v", env.zombie.Health)
	}
}

// Test 9c – merge into a non-compound path must be rejected
func TestDataModifyMergeValueNonCompound(t *testing.T) {
	env := newDataEnv(t)
	r, err := env.exec("data modify entity @n Health merge value {X:1b}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 0 {
		t.Fatalf("want success=0 (Health is float, not compound) got %d", r.Success)
	}
}

// Test 10 – data modify entity @n Health set from entity @n Health (copy from self)
func TestDataModifySetFrom(t *testing.T) {
	env := newDataEnv(t)
	before := env.zombie.Health
	r, err := env.exec("data modify entity @n Health set from entity @n Health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 1 {
		t.Fatalf("want success=1 got %d", r.Success)
	}
	if env.zombie.Health != before {
		t.Fatalf("Health changed unexpectedly: want %v got %v", before, env.zombie.Health)
	}
}

// Test 11 – player as write target (@s) must be rejected
func TestDataWriteTargetPlayerRejected(t *testing.T) {
	env := newDataEnv(t)
	if err := env.world.AddPlayer(env.player, "minecraft:overworld"); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	r, err := env.exec("data modify entity @s Health set value 5f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Success != 0 {
		t.Fatalf("want success=0 (player writes forbidden) got %d", r.Success)
	}
}
