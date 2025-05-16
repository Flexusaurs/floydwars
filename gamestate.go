package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type PlayerTurn int

type CardClass string

const (
	SOLDIER CardClass = "Soldier"
	MECH    CardClass = "Mech"
	Mage    CardClass = "Mage"
	Item    CardClass = "Item"
)

type Phase string

const (
	DrawPhase       Phase = "Draw"
	MainPhase       Phase = "Main"
	AttackPhase     Phase = "Attack"
	ResolutionPhase Phase = "Resolution"
	EndPhase        Phase = "End"
)

type CardEntity struct {
	id          string
	title       string
	dmg         int
	description string
	destroyed   bool
	skill       string
	CardClass   CardClass
}

type PlayerEntity struct {
	ID        string
	hand      []CardEntity
	itemCards []CardEntity
}

type GameState struct {
	Players       map[string]*PlayerEntity
	TurnPlayerID  string
	Phase         Phase
	PendingAttack *AttackContext
	FighterDeck   []CardEntity
	ItemDeck      []CardEntity
}

type AttackContext struct {
	AttackerID     string
	TargetID       string
	AttackerItem   *CardEntity
	DefenderItem   *CardEntity
	DamageResolved bool
}

func findCardByID(cards []CardEntity, id string) (*CardEntity, error) {
	for i := range cards {
		if cards[i].id == id {
			return &cards[i], nil
		}
	}
	return nil, errors.New("card not found")
}

func findCardsByIDs(cards []CardEntity, ids []string) ([]CardEntity, error) {
	var results []CardEntity
	for _, id := range ids {
		card, err := findCardByID(cards, id)
		if err != nil {
			return nil, err
		}
		results = append(results, *card)
	}
	return results, nil
}

func removeCard(hand []CardEntity, id string) []CardEntity {
	for i, card := range hand {
		if card.id == id {
			return append(hand[:i], hand[i+1:]...)
		}
	}
	return hand
}

func (p *PlayerEntity) ShuffleHand() {
	rand.NewSource(time.Now().UnixNano())
	n := len(p.hand)
	for i := range p.hand {
		j := rand.Intn(n)
		p.hand[i], p.hand[j] = p.hand[j], p.hand[i]
	}
}

func (p *PlayerEntity) ShuffleAndRedrawHand(gs *GameState) {
	count := len(p.hand)
	p.ShuffleHand()
	p.hand = []CardEntity{} // clear
	for i := 0; i < count; i++ {
		card, err := DrawFighterCard(gs, p)
		if err == nil && card != nil {
			p.hand = append(p.hand, *card)
		}
	}
}

func getOpponent(gs *GameState, currentID string) *PlayerEntity {
	for id, p := range gs.Players {
		if id != currentID {
			return p
		}
	}
	return nil
}

// Helper to find a card and its owner in the game state
func findCardByIDInGame(gs *GameState, cardID string) (*CardEntity, *PlayerEntity, error) {
	for _, player := range gs.Players {
		for i := range player.hand {
			if player.hand[i].id == cardID {
				return &player.hand[i], player, nil
			}
		}
		for i := range player.itemCards {
			if player.itemCards[i].id == cardID {
				return &player.itemCards[i], player, nil
			}
		}
	}
	return nil, nil, errors.New("card not found in game")
}

func StartAttack(gs *GameState, input AttackContext) error {
	if gs.Phase != MainPhase {
		return errors.New("you can only attack during the main phase")
	}

	// Validate attacker and target, store pending attack
	gs.PendingAttack = &AttackContext{
		AttackerID: input.AttackerID,
		TargetID:   input.TargetID,
	}
	gs.Phase = AttackPhase
	return nil
}

func DefenderUseItem(gs *GameState, item *CardEntity) error {
	if gs.Phase != AttackPhase {
		return errors.New("items can only be used during attack phase")
	}
	if gs.PendingAttack == nil {
		return errors.New("no pending attack")
	}
	gs.PendingAttack.DefenderItem = item
	return nil
}

func AttackerUseItem(gs *GameState, item *CardEntity) error {
	if gs.Phase != AttackPhase {
		return errors.New("items can only be used during attack phase")
	}
	if gs.PendingAttack == nil || gs.PendingAttack.DefenderItem == nil {
		return errors.New("attacker can only use item after defender used one")
	}
	gs.PendingAttack.AttackerItem = item
	return nil
}

func canAttackerDestroy(attacker, target *CardEntity) bool {
	// This logic can be extended to support AoE or multi-target later
	return attacker.dmg >= target.dmg
}

func applyItemEffects(item *CardEntity, owner *CardEntity, opponent *CardEntity) error {
	switch item.title {
	case "Armor Buff":
		owner.dmg += 1
	case "Weakening Shot":
		opponent.dmg -= 1
	case "Shield Bubble":
		// Prevent this card from being destroyed this turn
		owner.skill = "ImmuneThisTurn" // You can also set a flag
	case "Mirror Shield":
		// Redirects the attack to another card maybe? Up to you.
	default:
		return errors.New("unknown item effect: " + item.title)
	}
	return nil
}

func resolveCardSkills(card *CardEntity, trigger string, gs *GameState) {
	if card.skill == "" {
		return
	}

	ownerID := cardOwnerID(gs, card.id)
	player := gs.Players[ownerID]

	switch card.skill {
	case "ShuffleHand":
		player.ShuffleHand()
	case "Draw2":
		DrawFighterCard(gs, player)
		// Add more skills here
	}
}

// cardOwnerID returns the ID of the player who owns the card with the given ID.
func cardOwnerID(gs *GameState, cardID string) string {
	for id, player := range gs.Players {
		for _, c := range player.hand {
			if c.id == cardID {
				return id
			}
		}
		for _, c := range player.itemCards {
			if c.id == cardID {
				return id
			}
		}
	}
	return ""
}

func ResolveAttack(gs *GameState) error {
	if gs.Phase != AttackPhase || gs.PendingAttack == nil {
		return errors.New("no attack to resolve")
	}

	ctx := gs.PendingAttack

	attacker, attackerOwner, err := findCardByIDInGame(gs, ctx.AttackerID)
	if err != nil {
		return err
	}

	target, targetOwner, err := findCardByIDInGame(gs, ctx.TargetID)
	if err != nil {
		return err
	}

	// Apply item effects (modifies attacker/target stats if needed)
	if ctx.DefenderItem != nil {
		err := applyItemEffects(ctx.DefenderItem, target, attacker)
		if err != nil {
			return err
		}
		// Remove from itemCards
		targetOwner.itemCards = removeCard(targetOwner.itemCards, ctx.DefenderItem.id)
	}

	if ctx.AttackerItem != nil {
		err := applyItemEffects(ctx.AttackerItem, attacker, target)
		if err != nil {
			return err
		}
		attackerOwner.itemCards = removeCard(attackerOwner.itemCards, ctx.AttackerItem.id)
	}

	// --- Combat Resolution ---
	canDestroy := canAttackerDestroy(attacker, target)

	if canDestroy {
		target.destroyed = true
		targetOwner.hand = removeCard(targetOwner.hand, target.id)
		resolveCardSkills(target, "onDestroy", gs)
	}

	// Attacker is discarded after attack
	attackerOwner.hand = removeCard(attackerOwner.hand, attacker.id)

	// Cleanup and advance phase
	gs.PendingAttack = nil
	gs.Phase = ResolutionPhase
	return nil
}

func ResolveCombat(gs *GameState, input AttackContext) error {
	attackerPlayer := gs.Players[gs.TurnPlayerID]
	defenderPlayer := getOpponent(gs, gs.TurnPlayerID)

	attackerCard, err := findCardByID(attackerPlayer.hand, input.AttackerID)
	if err != nil {
		return fmt.Errorf("attacker card not found:%w", err)
	}

	targetCard, err := findCardByID(defenderPlayer.hand, input.TargetID)
	if err != nil {
		return fmt.Errorf("target card not found: %w", err)
	}

	attackdmg := attackerCard.dmg

	if input.DefenderItem != nil {
		attackdmg -= input.DefenderItem.dmg
	}

	if attackdmg < targetCard.dmg {
		return errors.New("not enough damage to defeat target")
	}

	attackerPlayer.hand = removeCard(attackerPlayer.hand, attackerCard.id)
	defenderPlayer.hand = removeCard(defenderPlayer.hand, targetCard.id)

	if input.AttackerItem != nil {
		attackerPlayer.hand = removeCard(attackerPlayer.hand, input.AttackerItem.id)
	}
	if input.DefenderItem != nil {
		defenderPlayer.hand = removeCard(defenderPlayer.hand, input.DefenderItem.id)
	}

	return nil
}

// Draws a fighter card from the shared deck
func DrawFighterCard(gs *GameState, p *PlayerEntity) (*CardEntity, error) {
	if len(gs.FighterDeck) == 0 {
		return nil, errors.New("fighter deck is empty")
	}
	card := gs.FighterDeck[0]
	gs.FighterDeck = gs.FighterDeck[1:]
	p.hand = append(p.hand, card)
	return &card, nil
}

// Draws an item card from the shared deck
func DrawItemCard(gs *GameState, p *PlayerEntity) (*CardEntity, error) {
	if len(gs.ItemDeck) == 0 {
		return nil, errors.New("item deck is empty")
	}
	card := gs.ItemDeck[0]
	gs.ItemDeck = gs.ItemDeck[1:]
	p.itemCards = append(p.itemCards, card)
	return &card, nil
}
