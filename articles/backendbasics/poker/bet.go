package poker

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Player struct {
	Name         string
	Cash         int
	Cards        [2]Card
	Folded       bool
	BetThisRound int  // amount bet this round
	AllIn        bool // true if the player has gone all-in
}
type Round byte

const (
	PreFlop Round = iota
	Flop
	Turn
	River
)

func (r Round) String() string {
	switch r {
	case PreFlop:
		return "PreFlop"
	case Flop:
		return "Flop"
	case Turn:
		return "Turn"
	case River:
		return "River"
	default:
		return "Unknown"
	}
}

type Game struct {
	rng       *rand.Rand
	players   []Player
	community [5]Card // community cards
	round     Round

	position byte // index of the player whose turn it is; < len(players)
	blind    byte // index of the player who is the big blind;  < len(players)

	currentBet int // current amount to call; 0 if no bet
	pot        int // total amount of money in the pot
	smallBlind int // current blind rate.

	deck Deck

	// buf holds intermediate state for resolving hands,
	// so we don't have to allocate between hands.
	buf struct {
		stillIn [8]byte
		winners [8]byte
		hands   [8]Hand
	}
}

const startingSmallBlind = 10
const startingCash = 1000
const blindIncreasesEvery = 10 // blind increases every N hands
const blindIncreasesBy = 10    // blind increases by N

type ActionKind byte

const (
	FOLD       ActionKind = iota // fold and forfeit the pot.
	CHECK_CALL                   // check or call the current bet.
	RAISE                        // raise by the amount in Action.Amount (which must be at least the current bet)
	ALLIN                        // go all-in with the rest of your money
)

// TakeAction attempts to take the given action for the given player. It does NOT advance the game; do that on a nil error.
func TakeAction(g *Game, player string, action ActionKind, amount int) error {
	if player != g.players[g.position].Name {
		return fmt.Errorf("it is not %q's turn", player)
	}
	var needToBet int
	switch action {
	// you can always fold
	case ALLIN:
		log.Printf("player %q goes all-in for %d", player, g.players[g.position].Cash)
		amount = g.players[g.position].Cash
		g.players[g.position].Cash = 0
		g.pot += amount
		g.players[g.position].BetThisRound += amount
		g.currentBet = max(g.currentBet, amount)
		g.players[g.position].AllIn = true
		return nil

	case FOLD:
		g.players[g.position].Folded = true
		return nil
	case CHECK_CALL:
		needToBet = g.currentBet - g.players[g.position].BetThisRound
		if needToBet > g.players[g.position].Cash { // go all-in if you don't have enough money to call
			return TakeAction(g, player, ALLIN, 0)
		}

		// otherwise, call / check the current bet and mark this player as having satisfied the bet this round
		if g.currentBet == 0 {
			log.Printf("player %q checks", player)
		} else {
			log.Printf("player %q calls for %d", player, g.currentBet)
		}

		g.players[g.position].Cash -= needToBet
		g.players[g.position].BetThisRound = g.currentBet
		return nil
	case RAISE:
		// if you don't have enough money to raise, you can use all of your money to raise by going all-in
		if g.currentBet >= g.players[g.position].Cash {
			return TakeAction(g, player, ALLIN, 0)
		}
		if amount < g.currentBet*2 {
			return fmt.Errorf("amount %d is less than twice the current bet: cannot raise without going all-in", player, amount, g.currentBet)
		}
		// otherwise, raise by the given amount
		g.currentBet = amount
		log.Printf("player %q raises to %d", player, amount)
		g.players[g.position].Cash -= (amount - g.players[g.position].BetThisRound)
		return nil
	default:
		return fmt.Errorf("invalid action kind %#+v", action)
	}
}

// removeBustedPlayers removes players who don't have enough money to pay the big blind.
func removeBustedPlayers(p []Player, smallBlind int) (stayed, left []Player) {
	bigBlind := smallBlind * 2
	startingLen := len(p)
	for i := len(p) - 1; i >= 0; i-- {
		if p[i].Cash < bigBlind {
			log.Printf("player %q busted out", p[i].Name)
			p[i], p[len(p)-1] = p[len(p)-1], p[i]
			p = p[:len(p)-1]
		}
	}
	return p, p[len(p):startingLen]
}

// NewGame returns a new game with the given players and small blind.
func NewGame(playerNames []string, smallBlind int) *Game {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	players := make([]Player, len(playerNames))
	for i := range players {
		players[i] = Player{
			Name: playerNames[i],
			Cash: startingCash,
		}
	}
	rng.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	return &Game{
		rng:        rng,
		players:    players,
		smallBlind: smallBlind,
		deck:       NewDeck(),
	}
}

type Action struct {
	Kind   ActionKind
	Amount int // only used for ActionKind == Raise
	Player string
}

func Run(players []string, actions <-chan Action) (winner string, err error) {
	g := NewGame(players, startingSmallBlind)

	for hand := 0; ; hand++ {
		// ----- housekeeping ----
		var removed []Player
		g.players, removed = removeBustedPlayers(g.players, g.smallBlind)
		for _, p := range removed {
			log.Printf("player %q busted out. better luck next time!", p.Name)
		}
		if len(g.players) == 1 { // only one player left; they win
			return g.players[0].Name, nil
		}
		if hand%blindIncreasesEvery == 0 { // increase the blinds every N hands
			g.smallBlind += blindIncreasesBy
			log.Printf("blinds increased to %d", g.smallBlind)
		}

		// cleanup the state from the previous hand
		g.pot, g.currentBet = 0, 0
		g.round = PreFlop
		g.rng.Shuffle(g.deck.Len(), func(i, j int) { g.deck.Swap(i, j) })

		g.blind = (g.blind + 1) % byte(len(g.players)) // small blind moves forward

		g.currentBet = 2 * g.smallBlind // the big blind is the first bet of the hand

		g.players[g.blind].Cash -= g.smallBlind // small blind must pay
		g.players[g.blind].BetThisRound = g.smallBlind

		g.players[(g.blind+1)%byte(len(g.players))].Cash -= g.smallBlind * 2 // big blind must pay
		g.players[(g.blind+1)%byte(len(g.players))].BetThisRound = g.smallBlind * 2

		for {
			// find the next player who hasn't folded
			for i := range g.players {
				if g.players[i].Folded || g.players[i].AllIn {
					continue
				}
				if g.players[i].BetThisRound < g.currentBet {
					action := <-actions
					if err := TakeAction(g, action.Player, action.Kind, action.Amount); err != nil {
						log.Printf("error taking action: %v", err)
					}

				}
			}
		}
		g.position = (g.blind + 2) % byte(len(g.players)) // the player after the big blind goes first

	}
}

// resolveHand resolves the current hand, giving the pot to the best hand.
func (g *Game) resolveHand() {
	stillIn := g.buf.stillIn[:0]
	for i := range g.players {
		if g.players[i].Folded {
			continue
		}
		stillIn = append(stillIn, byte(i))
	}
	log.Printf("resolving hand... %d players left", len(stillIn))

	switch len(stillIn) {
	case 0: // no one left; no winner; should never happen
		return
	case 1: // one player left; they win
		g.players[stillIn[0]].Cash += g.pot
		log.Printf("player %q takes a pot worth %d", g.players[stillIn[0]].Name, g.pot)
		g.pot = 0
		return
	}

	hands := g.buf.hands[:len(stillIn)]
	var bestHand Hand
	for i, j := range stillIn {
		c := g.players[j].Cards
		hands[i] = GetHand(c[0], c[1], &g.community)
		if hands[i].Greater(bestHand) {
			bestHand = hands[i]
		}
	}
	winners := g.buf.winners[:0]
	for i, j := range stillIn {
		if hands[i] == bestHand {
			winners = append(winners, j)
		}
	}
	switch len(winners) {
	case 0: // no one won; should never happen
		panic("no one won: this should never happen!")
	case 1: // one winner; they take the pot
		g.players[winners[0]].Cash += g.pot
		log.Printf("player %q takes a pot worth %d", g.players[winners[0]].Name, g.pot)
		g.pot = 0
		return
	default:
		// time to split the pot among the winners. TODO: figure out the logic for splits on all-ins, etc.
		// for now, we'll just split it evenly among the winners.

		payout := g.pot / len(winners)
		for _, i := range winners {
			g.players[i].Cash += payout
			g.pot -= payout
			log.Printf("player %q takes a 1/%d share of the pot worth %d", g.players[i].Name, len(winners), payout)
		}
		if g.pot > 0 {
			log.Printf("house takes the remainder %d", g.pot)
			g.pot = 0
		}
		return

	}
}

// bestHand returns the best hand of any remaining player.
func bestHand(players []Player, shared *[5]Card) Hand {

	var best Hand
	for i := range players {
		hand := GetHand(players[i].Cards[0], players[i].Cards[1], shared)
		if hand.Greater(best) {
			best = hand
		}
	}
	return best
}
