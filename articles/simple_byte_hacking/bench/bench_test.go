package bench_test

import (
	"encoding/hex"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// go's compiler will try and optimize out functions that don't do anything,
// so we attempt to work around that by assigning to this global.
var _uid uuid.UUID

// in TestMain, we generate 128 random UUIDs and alternate which one we use for each test,
// just in case UUIDs take different amounts of time to format or parse
// depending on their format. this may be overly clever.
var (
	uuids       [128]uuid.UUID
	uuidStrings [128]string
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	for i := range uuids {
		uuids[i] = uuid.New()
	}
	// preformat the UUIDs as strings for the "string" scenario
	for i := range uuidStrings {
		uuidStrings[i] = uuids[i].String()
	}
	os.Exit(m.Run())
}

func TestSwapEquivalence(t *testing.T) {
	for i := 0; i < 20000; i++ {
		u := uuid.New()
		res8, res16, resDirect, resJS := swap8(u), swap16(u), swapDirect(u), swapJS(u.String())
		if res8 != res16 || res8 != resDirect || res8 != resJS {
			t.Errorf("expected all 4 to be equivalent: %s %s %s %s", res8, res16, resDirect, resJS)
		}
		if swap8(res8) != u || swap16(res16) != u || swapDirect(resDirect) != u || (swapJS(resJS.String())) != u {
			t.Errorf("expected swap to be it's own inverse: %s %s %s %s", res8, res16, resDirect, resJS)
		}
	}
}

func BenchmarkSwap8(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swap8(uuids[b.N%128])
	}
}

func BenchmarkSwap8String(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swap8(uuid.MustParse(uuidStrings[b.N%128]))
	}
}

func BenchmarkSwap16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swap16(uuids[b.N%128])
	}
}

func BenchmarkSwap16String(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swap16(uuid.MustParse(uuidStrings[b.N%128]))
	}
}

func BenchmarkSwapDirect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swapDirect((uuids[b.N%128]))
	}
}

func BenchmarkSwapDirectString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swapDirect(uuid.MustParse(uuidStrings[b.N%128]))
	}
}

func BenchmarkSwapJS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swapJS((uuids[b.N%128].String()))
	}
}

func BenchmarkSwapJSString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_uid = swapJS((uuidStrings[b.N%128]))
	}
}

var (
	tbl8  = [8]byte{3, 2, 1, 0, 5, 4, 7, 6}
	tbl16 = [16]byte{3, 2, 1, 0, 5, 4, 7, 6, 8, 9, 10, 11, 12, 13, 14, 15}
)

func swap8(src uuid.UUID) uuid.UUID {
	dst := src               // copy the entire thing
	for i, j := range tbl8 { // and permute the first half according to the swap table.
		dst[i] = src[j]
	}
	return dst
}

func swap16(src uuid.UUID) uuid.UUID {
	var dst uuid.UUID
	for i, j := range tbl16 { // permute according to the table.
		dst[i] = src[j]
	}
	return dst
}

var removeExtras = strings.NewReplacer("{", "", "}", "", "-", "")

func swapDirect(u uuid.UUID) uuid.UUID {
	u[0], u[1], u[2], u[3] = u[3], u[2], u[1], u[0]
	u[4], u[5] = u[5], u[4]
	u[6], u[7] = u[7], u[6]
	return u
}

func swapJS(id string) uuid.UUID {
	b16 := removeExtras.Replace(id)
	a := b16[6:6+2] + b16[4:4+2] + b16[2:2+2] + b16[0:0+2]
	b := b16[10:10+2] + b16[8:8+2]
	c := b16[14:14+2] + b16[12:12+2]
	d := b16[16:]
	src, _ := hex.DecodeString(a + b + c + d)
	var dst uuid.UUID
	copy(dst[:], src)
	return dst
}
