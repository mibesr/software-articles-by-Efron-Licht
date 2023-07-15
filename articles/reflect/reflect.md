THIS IS UNFINISHED AND UNPUBLISHED.
READ AT YOUR OWN RISK.

# reflect

A software article by Efron Licht
July 2023

## Our goals

Arbitrary manipulation of gamestate at runtime with a forgiving API. Ideally, the combination of console and other debug tools should mean there's no difference between 'playing' the game and 'building/debugging' the game.

High-level goals:

- modify arbitray gamestate at runtime
  - change variables
  - call functions, both built-in to the console or available by walking the gamestate DAG
- UI is also gamestate
  - modify UI live
  - connect variables to player input (mouse position, keyboard input, etc)

## performance

'idle' performance should be as close to zero as possible. The console should not affect performance unless it is being used.

the console should start up _instantly_ and not affect the game's startup time: hard max of 5ms. doing a eager-nonblocking load and not letting the console open on the first frame is totally fine.

commands should be reasonably performant, but since they're only triggered on player input, they're not nearly as big as a deal, since even with a human w/ frame-perfect input, you still can't input a command more than a couple times a second.

OTOH, any 'callback' behavior that triggers on gamestate changes should be as fast as possible, since it can be triggered arbitrarily often. it is OK for debugging tools to update less often than the game does: the console should NEVER drop a frame except immediately after processing a new command.

## example uses of the console

## modify UI live

## 'cheat codes': 
- player.hp = 1000000
## change gun defintions live for NPCs and players

### enable or disable UI elements

- toggle ui.healthbar

## connect variables to player input (mouse position, keyboard input, etc)

- `followmouse ui.healthbar`
- `followmouse player`

|symbol|name|unicode|
|---|---|---|
|↑|up arrow|U+2191|
|↓|down arrow|U+2193|
|←|left arrow|U+2190|
|→|right arrow|U+2192|
|⇧|shift|U+21E7|
|⌃|control|U+2303|

### console UI

traditional console UI: command history, autocomplete, etc. history is saved to disk.

|mod|keys|action|note|
|---|---|---|---|
||`↑/↓`|scroll through command history|
||`tab`|autocomplete|
||`←/→`|move cursor|
||`␈`|backspace|
|`⇧`| `←/→`|move cursor by word|
|`⌃`|`←/→`|move cursor to beginning/end of line|
|`⇧`|`␈`|delete word|
|`⌃`|`␈`|delete line|
|`⌃`|`c`| copy line to system keyboard|
|`⌃`|`v`| paste line to system keyboard|

### forgiving API

- console should never break the game entirely: all changes should be caught by `reset` at very least
- panics are caught and logged: console should never crash the game
- no worrying about capitalization: unexported fields should be invisible
  - warn on name conflicts at runtime.
- autocomplete should suggest fields and methods
- easy access to array and slice fields
  - combine LUA-like and Python-like index syntax:
  - `npcs.0` <==> `npcs[0]`
  - `npcs.-1` <==> `npcs[len(npcs)-1]`
  - can go back and forth between struct-like, map-like, and array-like access:
    suppose `npcs = []struct{foo map[string]int}: npcs.0.foo.bar <==>`npcs[0].foo["bar"]`
  - user should not have to consider pointer vs value, slice vs array
  - automatic type coersions
    - numbers: use c-like rules, casting to the widest type to do math and then truncating back to the original type. numerical error, etc, is OK: this is meant for debugging, not truth.
    - cast strings to and from other types as needed: other types can be ad-hoc made from strings via TextUnmarshaler
- prefer `toggle ui.healthbar` over `ui.healthbar.enabled = !ui.healthbar.enabled`
- prefer `followmouse ui.healthbar` over `follow(ui.healthbar, mouse)`
- automatically handle pointer ref/deref where possible: console should have no idea what a pointer is: everything shoudl be OK for [`reflect.Value.CanAddr()`](https://pkg.go.dev/reflect#Value.CanAddr).
-
