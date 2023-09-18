// Package poker implements a simple texas hold'em poker game.
package poker

import (
	"fmt"
	"math/rand"
	"strings"
)

// Rank is a playing card rank, from Ace to King.
type Rank byte

// Suit is a playing card suit, from Clubs to Spades.
type Suit byte

// Card is a playing card made up of a rank and a suit.
type Card struct {
	Rank
	Suit
}

// Shuffle the deck using the provided rng.
func (d *Deck) Shuffle(rng *rand.Rand) { rng.Shuffle(d.Len(), d.Swap) }

// NewDeck returns a new, unshuffled deck of cards.
func NewDeck() Deck {
	var d Deck
	var i int
	for s := Suit(1); s < SuitMax; s++ {
		for r := Rank(1); r < RankMax; r++ {
			d[i] = Card{r, s}
			i++
		}
	}

	return d
}

// Equal returns true if the two hands are of the same kind and have the same high and low cards.
func (h Hand) Equal(o Hand) bool {
	if h.Kind != o.Kind {
		return false
	}
	switch h.Kind {
	case TwoPair, FullHouse:
		return h.High == o.High && h.Low == o.Low
	default:
		return h.High == o.High
	}
}

func (h Hand) Greater(o Hand) bool {
	if h == o {
		return false
	}
	return !h.Less(o)
}

// Less returns true if h is a worse hand than o.
func (h Hand) Less(o Hand) bool {
	if (h == Hand{}) && (o != Hand{}) {
		return true
	}
	switch {
	case h.Kind < o.Kind:
		return true
	case h.Kind > o.Kind:
		return false
	case h.High < o.High:
		return true
	case h.High > o.High:
		return false
	case h.Kind == TwoPair || h.Kind == FullHouse:
		return h.Low < o.Low
	default:
		return false // no way to decide
	}
}

// Less orders cards. Aces are high; suits are ordered alphabetically.
func (c Card) Less(o Card) bool {
	if c.Rank == o.Rank {
		return c.Suit < o.Suit
	}
	if c.Rank == Ace {
		return false
	}
	return c.Rank < o.Rank
}

func (c *Card) MarshalJSON() ([]byte, error) { return []byte(fmt.Sprintf("%q", c.Notation())), nil }
func (c *Card) UnmarshalJSON(b []byte) error {
	card, ok := CardFromNotation(strings.Trim(string(b), `"`))
	if !ok {
		return fmt.Errorf("invalid card notation: %q", b)
	}
	*c = card
	return nil
}

func (c *Card) MarshalText() ([]byte, error) { return []byte(c.Notation()), nil }
func (c *Card) UnmarshalText(b []byte) error {
	card, ok := CardFromNotation(string(b))
	if !ok {
		return fmt.Errorf("invalid card notation: %q", b)
	}
	*c = card
	return nil
}

// Notation returns the card's notation, e.g. "AC" for Ace of Clubs. Compare to String().
func (c Card) Notation() string { return notation[c.Rank][c.Suit] }

var notation = [RankMax][SuitMax]string{
	UNKNOWN: {"??", "??", "??", "??"},
	Ace:     {"AC", "AD", "AH", "AS"},
	Two:     {"2C", "2D", "2H", "2S"},
	Three:   {"3C", "3D", "3H", "3S"},
	Four:    {"4C", "4D", "4H", "4S"},
	Five:    {"5C", "5D", "5H", "5S"},
	Six:     {"6C", "6D", "6H", "6S"},
	Seven:   {"7C", "7D", "7H", "7S"},
	Eight:   {"8C", "8D", "8H", "8S"},
	Nine:    {"9C", "9D", "9H", "9S"},
	Ten:     {"TC", "TD", "TH", "TS"},
	Jack:    {"JC", "JD", "JH", "JS"},
	Queen:   {"QC", "QD", "QH", "QS"},
	King:    {"KC", "KD", "KH", "KS"},
}

// CardFromString parses a card from either it's formal name ("Ace of Clubs") or its notation ("AC").
func CardFromString(s string) (Card, bool) {
	if len(s) == 2 {
		return CardFromNotation(s)
	}
	return CardFromName(s)
}

func CardFromName(s string) (Card, bool) {
	rawRank, rawSuit, ok := strings.Cut(s, " of ")
	if !ok {
		return Card{}, false
	}
	var rank Rank
	switch rawRank {
	case "Ace", "ace", "A", "a":
		rank = Ace
	case "Two", "two", "2":
		rank = Two
	case "Three", "three", "3":
		rank = Three
	case "Four", "four", "4":
		rank = Four
	case "Five", "five", "5":
		rank = Five
	case "Six", "six", "6":
		rank = Six
	case "Seven", "seven", "7":
		rank = Seven
	case "Eight", "eight", "8":
		rank = Eight
	case "Nine", "nine", "9":
		rank = Nine
	case "Ten", "ten", "T", "t", "10":
		rank = Ten
	case "Jack", "jack", "J", "j":
		rank = Jack
	case "Queen", "queen", "Q", "q":
		rank = Queen
	case "King", "king", "K", "k":
		rank = King
	default:
		return Card{}, false
	}

	var suit Suit
	switch rawSuit {
	case "Clubs", "clubs", "C", "c":
		suit = Clubs
	case "Diamonds", "diamonds", "D", "d":
		suit = Diamonds
	case "Hearts", "hearts", "H", "h":
		suit = Hearts
	case "Spades", "spades", "S", "s":
		suit = Spades
	default:
		return Card{}, false
	}
	return Card{rank, suit}, true
}

// CardNotation parses a card from its notation, e.g. "AC" for Ace of Clubs.
func CardFromNotation(s string) (Card, bool) {
	if len(s) != 2 {
		return Card{}, false
	}
	var rank Rank
	var suit Suit
	switch s[0] {
	case 'A':
		rank = Ace
	case '2':
		rank = Two
	case '3':
		rank = Three
	case '4':
		rank = Four
	case '5':
		rank = Five
	case '6':
		rank = Six
	case '7':
		rank = Seven
	case '8':
		rank = Eight
	case '9':
		rank = Nine
	case 'T':
		rank = Ten
	case 'J':
		rank = Jack
	case 'Q':
		rank = Queen
	case 'K':
		rank = King
	default:
		return Card{}, false
	}
	switch s[1] {
	case 'C':
		suit = Clubs
	case 'D':
		suit = Diamonds
	case 'H':
		suit = Hearts
	case 'S':
		suit = Spades
	default:
		return Card{}, false
	}
	return Card{rank, suit}, true
}

const UNKNOWN = 0

// Ranks are Ace, Two, Three, ..., Queen, King.
const (
	Ace Rank = iota + 1 // Ace is both high and low
	Two
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack    // 11
	Queen   // 12
	King    // 13
	RankMax // must be last
)

// Suits are Clubs, Diamonds, Hearts, Spades, ordered alphabetically.
const (
	Clubs Suit = iota + 1
	Diamonds
	Hearts
	Spades
	SuitMax // SuitMax must be last
)

type HandKind byte

// Hand kinds, in order from lowest to highest.
const (
	HighCard HandKind = iota
	Pair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	HandKindMax
)

func (r Rank) String() string { return rankNames[r] }
func (s Suit) String() string { return suitNames[s] }

// String formats a card as "<rank> of <suit>".
func (c Card) String() string { return fmt.Sprintf("%s of %s", c.Rank, c.Suit) }

var (
	rankNames = [RankMax]string{
		UNKNOWN: "Unknown Rank",
		Ace:     "Ace",
		Two:     "Two",
		Three:   "Three",
		Four:    "Four",
		Five:    "Five",
		Six:     "Six",
		Seven:   "Seven",
		Eight:   "Eight",
		Nine:    "Nine",
		Ten:     "Ten",
		Jack:    "Jack",
		Queen:   "Queen",
		King:    "King",
	}

	kindNames = [HandKindMax]string{
		HighCard:      "High Card",
		Pair:          "Pair",
		TwoPair:       "Two Pair",
		ThreeOfAKind:  "Three of a Kind",
		Straight:      "Straight",
		Flush:         "Flush",
		FullHouse:     "Full House",
		FourOfAKind:   "Four of a Kind",
		StraightFlush: "Straight Flush",
	}

	suitNames = [SuitMax]string{
		UNKNOWN:  "Unknown Suit",
		Spades:   "Spades",
		Hearts:   "Hearts",
		Diamonds: "Diamonds",
		Clubs:    "Clubs",
	}
)

func (k HandKind) String() string { return kindNames[k] }

// Deck is a standard 52-card deck of playing cards.
type Deck [52]Card

func (d *Deck) Len() int { return len(d) }
func (d *Deck) Less(i, j int) bool {
	if d[i].Rank == d[j].Rank {
		return d[i].Suit < d[j].Suit
	}
	return d[i].Rank < d[j].Rank
}
func (d *Deck) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

type Hand struct {
	Kind HandKind // kind of hand; e.g. Flush
	High Rank     // highest scoring card; e.g, if we have a full house, this is the rank of the three-of-a-kind
	Low  Rank     // lowest scoring card; e.g, if we have two pair, this is the lower pair's rank
}

func (h Hand) String() string {
	switch h.Kind {
	case FullHouse, TwoPair:
		return fmt.Sprintf("%s (%s, %s)", h.Kind, h.High, h.Low)
	case HighCard, Pair, ThreeOfAKind, FourOfAKind:
		return fmt.Sprintf("%s (%s)", h.Kind, h.High)
	case Straight:
		return fmt.Sprintf("%s (%s high)", h.Kind, h.High)
	case StraightFlush, Flush:
		return fmt.Sprintf("%s (%s high)", h.Kind, h.High)
	default:
		return fmt.Sprintf("%#+v", h)
	}
}

// GetHand returns the best hand that can be made from the given cards.
// The first two cards are the player's "hole" cards, and the remaining
// five are the "shared" cards.
func GetHand(a, b Card, shared *[5]Card) Hand {
	cards := make([]Card, 7)
	copy(cards[:], shared[:])
	var ranks [RankMax]byte
	var suits [SuitMax]byte

	for _, c := range cards {
		ranks[c.Rank]++
		suits[c.Suit]++
	}

	var flush Suit

	for i := range suits {
		if suits[i] == 5 {
			flush = Suit(i)
			break
		}
	}
	var straight, fourOfAKind, threeOfAKind, pair, pair2, high Rank

	// check for rank-based hands; we'll choose the best one later
	for i := Five; i <= King; i++ {
		if ranks[i] > 0 && ranks[i-1] > 0 && ranks[i-2] > 0 && ranks[i-3] > 0 && ranks[i-4] > 0 {
			straight = i
		}
		switch ranks[i] {
		case 4:
			fourOfAKind = i
		case 3:
			threeOfAKind = i
		case 2:
			if pair != 0 {
				pair, pair2 = i, pair
			} else {
				pair = i
			}
		default:
			high = i
		}
	}
	// check for royal straight
	if ranks[Ace] > 0 && ranks[King] > 0 && ranks[Queen] > 0 && ranks[Jack] > 0 && ranks[Ten] > 0 {
		straight = Ace
	}
	if ranks[Ace] > 0 {
		high = Ace
	}

	// ok, now we know what kind of hand we have, and what the high card is
	// let's build the best hand we can

	switch {
	case straight != 0 && flush != 0:
		// is the straight the same as the flush?
		for _, c := range cards {
			if c.Suit == flush && c.Rank == straight {
				return Hand{StraightFlush, straight, 0}
			}
		}
		return Hand{Flush, high, 0}
	case fourOfAKind != 0:
		return Hand{FourOfAKind, fourOfAKind, high}
	case threeOfAKind != 0 && pair != 0:
		return Hand{FullHouse, threeOfAKind, pair}
	case flush != 0:
		return Hand{Flush, high, 0}
	case straight != 0:
		return Hand{Straight, straight, 0}
	case threeOfAKind != 0:
		return Hand{ThreeOfAKind, threeOfAKind, high}
	case pair != 0 && pair2 != 0:
		return Hand{TwoPair, pair, pair2}
	case pair != 0:
		return Hand{Pair, pair, high}
	default:
		return Hand{HighCard, high, 0}
	}
}
