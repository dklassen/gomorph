package gomorph_test

import (
	"fmt"
	"github.com/dklassen/gomorph"
	"reflect"
	"testing"
)

// RPG Character DTO and Model with more fields and different names/types
type CharacterDTO struct {
	Name      string
	Level     string
	HP        string
	IsNPC     string
	Class     string // e.g. "wizard"
	Race      string // e.g. "elf"
	Inventory string // comma-separated list
}

type CharacterModel struct {
	FullName  string
	Level     int
	HP        int
	IsNPC     bool
	CharClass CharacterClass
	Race      Race
	Items     []string
}

type CharacterClass string
type Race string

// --- TypeConverters ---

type StringToIntConverter struct{ gomorph.TypeMap[string, int] }

func (c StringToIntConverter) From(source any) (any, error) {
	s, ok := source.(string)
	if !ok {
		return 0, fmt.Errorf("expected string, got %T", source)
	}
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

type StringToBoolConverter struct{ gomorph.TypeMap[string, bool] }

func (c StringToBoolConverter) From(source any) (any, error) {
	s, ok := source.(string)
	if !ok {
		return false, fmt.Errorf("expected string, got %T", source)
	}
	switch s {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool string: %q", s)
	}
}

type StringToClassConverter struct {
	gomorph.TypeMap[string, CharacterClass]
}

func (c StringToClassConverter) From(source any) (any, error) {
	s, ok := source.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", source)
	}
	switch s {
	case "wizard", "mage":
		return CharacterClass("Wizard"), nil
	case "warrior":
		return CharacterClass("Warrior"), nil
	case "rogue":
		return CharacterClass("Rogue"), nil
	default:
		return "", fmt.Errorf("unknown class: %q", s)
	}
}

type StringToRaceConverter struct{ gomorph.TypeMap[string, Race] }

func (c StringToRaceConverter) From(source any) (any, error) {
	s, ok := source.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", source)
	}
	switch s {
	case "elf":
		return Race("Elf"), nil
	case "human":
		return Race("Human"), nil
	case "orc":
		return Race("Orc"), nil
	case "dwarf":
		return Race("Dwarf"), nil
	default:
		return "", fmt.Errorf("unknown race: %q", s)
	}
}

type CSVToSliceConverter struct {
	gomorph.TypeMap[string, []string]
}

func (c CSVToSliceConverter) From(source any) (any, error) {
	s, ok := source.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", source)
	}
	if s == "" {
		return []string{}, nil
	}
	return splitAndTrim(s, ","), nil
}
func splitAndTrim(s, sep string) []string {
	raw := make([]string, 0)
	for _, part := range split(s, sep) {
		raw = append(raw, trim(part))
	}
	return raw
}
func split(s, sep string) []string {
	var out []string
	start := 0
	for i := range s {
		if string(s[i]) == sep {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
func trim(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

// --- Validators ---

type LevelValidator struct{ gomorph.TypeMap[int, int] }

func (v LevelValidator) From(source any) (any, error) {
	i, ok := source.(int)
	if !ok {
		return 0, fmt.Errorf("expected int, got %T", source)
	}
	if i < 1 {
		return 0, fmt.Errorf("level must be >= 1")
	}
	return i, nil
}

type HPValidator struct{ gomorph.TypeMap[int, int] }

func (v HPValidator) From(source any) (any, error) {
	i, ok := source.(int)
	if !ok {
		return 0, fmt.Errorf("expected int, got %T", source)
	}
	if i <= 0 {
		return 0, fmt.Errorf("HP must be > 0")
	}
	return i, nil
}

func TestCharacterMapping_Complex(t *testing.T) {
	nameMapping := gomorph.From[string, string](gomorph.NewField[string]("Name")).
		To(gomorph.NewField[string]("FullName")).
		SkipConversion().SkipValidation().Build()

	levelMapping := gomorph.From[string, int](gomorph.NewField[string]("Level")).
		To(gomorph.NewField[int]("Level")).
		ConvertWith(StringToIntConverter{}).
		ValidateWith(LevelValidator{}).
		Build()

	hpMapping := gomorph.From[string, int](gomorph.NewField[string]("HP")).
		To(gomorph.NewField[int]("HP")).
		ConvertWith(StringToIntConverter{}).
		ValidateWith(HPValidator{}).
		Build()

	isNPCMapping := gomorph.From[string, bool](gomorph.NewField[string]("IsNPC")).
		To(gomorph.NewField[bool]("IsNPC")).
		ConvertWith(StringToBoolConverter{}).
		SkipValidation().
		Build()

	classMapping := gomorph.From[string, CharacterClass](gomorph.NewField[string]("Class")).
		To(gomorph.NewField[CharacterClass]("CharClass")).
		ConvertWith(StringToClassConverter{}).
		SkipValidation().
		Build()

	raceMapping := gomorph.From[string, Race](gomorph.NewField[string]("Race")).
		To(gomorph.NewField[Race]("Race")).
		ConvertWith(StringToRaceConverter{}).
		SkipValidation().
		Build()

	inventoryMapping := gomorph.From[string, []string](gomorph.NewField[string]("Inventory")).
		To(gomorph.NewField[[]string]("Items")).
		ConvertWith(CSVToSliceConverter{}).
		SkipValidation().
		Build()

	fieldMappers := []gomorph.FieldMapper{
		nameMapping, levelMapping, hpMapping, isNPCMapping,
		classMapping, raceMapping, inventoryMapping,
	}
	mapper := gomorph.NewStructMapper[CharacterDTO, CharacterModel](fieldMappers)

	dto := CharacterDTO{
		Name: "Gimli", Level: "12", HP: "85", IsNPC: "false",
		Class: "warrior", Race: "dwarf", Inventory: "axe, helmet, ale",
	}
	model, err := mapper.From(dto)
	if err != nil {
		t.Fatalf("mapping failed: %v", err)
	}
	expected := CharacterModel{
		FullName: "Gimli", Level: 12, HP: 85, IsNPC: false,
		CharClass: "Warrior", Race: "Dwarf",
		Items: []string{"axe", "helmet", "ale"},
	}
	if !reflect.DeepEqual(model, expected) {
		t.Errorf("expected %+v, got %+v", expected, model)
	}
}
