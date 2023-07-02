
# advanced go: reflection-based debug console

This article assumes an intermediate level of knowledge of Go, including at least the basics of reflection. You can learn about reflection via the [official docs on golang.org](https://golang.org/pkg/reflect/) or by reading [the laws of reflectionon the Go Blog].(<https://blog.golang.org/laws-of-reflection>).

## 1. Motivation

For the last few months I've been working nearly full-time on my own 2D Game, 'Tactical Tapir'. Working with games is a new domain for me: systems are not as isolable as they are in web development or device drivers. In some ways, the questions game development asks are much more subjective: instead of asking "is this code correct?", you ask "does this feel nice?"


Here's a non-exhaustive list of tasks I'd like to be able to do 'live':

- spawn an enemy or obstacle
- change player traits (health, movement speed, etc.)
- manipulate the positioning, size, or look of UI elements
- track the state of a variable or group of related variables from frame-to-frame.

## 2. An (incredibly brief) introduction to game development

A game is a program that runs in a loop. Each update, or `frame`, does the following:
- take player input
- update the state of the game usin
- 




While we might do parts within `Update()` in parallel, _between_ frames, the State is fixed. We're 'safe' to modify that state arbitarily, at the beginning of each frame, since we know that nothing else is touching it. And we're safe to read from it after Update(), since we know that nothing else is touching it.

That is, our game loop should look like this:

```go
for i := 0; ; i++ {
    inputs := input.ThisFrame()
    debugUpdate(game, inputs)
    if err := game.Update(inputs); err != nil {
        log.Fatalf("shutdown: update(): %v")
    }
    <-ticker // but wait for the next frame to draw
    if err := game.Draw(); err != nil {
        log.Fatalf("shutdown: draw(): %v")
    }
}
```

### 3: Naive cheats

Now that we know where to put them, let's write some cheats. Let's test the idea with two simple cheats: one to give the player infinite ammo, and one to give the player infinite health.

We'll use keyboard inputs to trigger the cheats. We don't want the player to trigger them accidentally, so we'll require that they hold down the `shift` and `ctrl` keys while pressing `H` and `A` respectively.


```go
func applyCheats(g *Game, input Inputs) {
    if input.Held[KeyShift] && input.Held[KeyCtrl] {// check for cheats: if no ctrl+shift, no cheats
        if input.JustPressed[KeyA] {
            log.Println("infinite ammo")
            g.Player.Ammo = math.Inf(1)
        }
        if input.JustPressed[KeyH] {
            log.Println("infinite health")
            g.Player.HP = math.Inf(1)
        }
    }
}
func (g *Game) Update(input Inputs) {
    applyCheats(g, input)
    // ...rest of update
}
```

This works pretty well. In fact, we can visualize it as a kind of table:
|cheat|key|description|
|---|---|---|
|∞ ammo|`ctrl`+`shift`+`A`|set `State.Player.Ammo` to math.Inf(1)|
|∞ hp|`ctrl`+`shift`+`H`|set `State.Player.HP` to math.Inf(1)|

Which naturally suggests using a map to store the cheats:

```go
var cheats map[Key]struct { 
    description string
    apply func(*Game)
} {
    KeyA: {
        description: "spawn ammo",
        apply: func(g *Game) {
            g.Pickups = append(g.Pickups, AmmoPickup{...})
        },
    },
    KeyH: {
        description: "spawn health",
        apply: func(g *Game) {
            g.Pickups = append(g.Pickups, HealthPickup{...})
        },
    },
}
func applyCheats(g *Game, input Inputs, cheats map[Key]struct {
    description string
    apply func(*Game)
}) {
    if input.Held[KeyShift] && input.Held[KeyCtrl] {// check for cheats: if no ctrl+shift, no cheats
        for key, cheat := range cheats { 
            if input.JustPressed[key]
                log.Println(cheat.description)
                cheat.f(g)
            }
    }
}

func (g *Game) Update(input Inputs) {
    applyCheats(g, input, cheats)
    // ...rest of update
}
```

While this works well for some tasks, a few limitations are immediately apparent:

- Every new field we touch requires it's own function, even if we have multiple fields that are logically related.
- Those functions must be niladic: that is, they don't take any arguments, they just modify the gamestate directly.
- So for each field, and for each value we want to set it to, we need a separate function.

For some things, like 'infinite HP', this is fine. But for other things, like 'exactly 28 HP', this is absurd. You'd either need to write a function for every possible value, or expand the system somehow to take arguments. We need a more general approach.

### 4: A more general approach
The default API for general-purpose inputs is the command line. That is, prompt that takes a line of text, parses it as a command, and executes it.

A basic API might look like this:
|command|description|example| example output |
|---|---|---|---|
| `<field> = <value>` | set the field to the value | `player.hp = 100` | player.hp -> 100
| `print <field>`| print the value of the field | `print player.hp` | player.hp: 100

Let's see how we might implement this. For now, I'm going to ignore rendering the prompt, and just focus on parsing the input.
```go
type Prompt struct {
    Cursor int
    Text string // current line of text.
}
func (p *Prompt) Update(input Input) (line string, pressedEnter bool) {
    if p.Cursor < 0 {
        p.Cursor = 0
    } else if p.Cursor > len(p.Text) {
        p.Cursor = len(p.Text)
    }

    switch {
        case input.JustPressed[KeyEnter] && p.Text != "": // return the current line of text
            line := p.Text
            p.Cursor, p.Text = 0, "" // 
            return line, true
        case input.JustPressed[KeyBackspace] && p.Cursor > 0: // delete the character before the cursor
            p.Text = p.Text[:p.Cursor-1] + p.Text[p.Cursor:]
            p.Cursor--
        for _, p := input.PrintableThisFrame()

    }
}
```
- Print a prompt to the screen that fills in with the user's input.
- Parse that input into a command when the user presses enter.
- Debug access should be opt-out, not opt-in.



The first game I played with a Debug Console was 1996's quake: TODO: PICTURE
And it's still a staple of modern games

- Factorio <TODO: PICTURE>
- Doom 2016  <TODO: PICTURE>

Many game developers inject a LUA interpreter for this reason. LUA is a fast & flexible language with an incredibly small runtime, and there may be pre-existing Go bindings for it - if you're in this situation, you may want to start there.

## 4: Reflection

## conceptual breakdown

We have four separate problems to solve.

- Resolving paths into a specific field of a struct or slice using reflection.
- Parsing user input into executable commands, usually in the form `op <path> <value>`
- Converting the user-provided value into the correct type.
- Setting the field to the new value.
Each of these problems will happen 'independently' for each command, so we can handle them independently.

Let's start with two basic commands: `set` and `print`. `set` will set a field to a value, and `print` will print the value of a field.

| op [args] | description | example|
|---|---|---|
| `set` | `set <path> <literal_or_path>` | `set player.health 100` |
| `print` | `print <path>` | `print player.health` |

Let's assume we have a complete line of user input:

```go
func resolvePath(root reflect.Value, path string) (v reflect.Value, err error){
    // as long as it's a *T or an interface{}, keep dereferencing it
    for root.Type().Kind() == reflect.Ptr  || root.Type().Kind() == reflect.Interface{
        root = root.Elem()
    }

    fields := strings.Split(path, ".")
    v := root
 
    for i, field := range fields {
        if root.Type().Kind() != reflect.Struct {
            return v, fmt.Errorf("%s.%s: expected struct, got %s", root, strings.Join(path[:i], "."), root.Type().Kind())
        }
        v = v.FieldByName(field)
        if !v.IsValid() {
            return v, fmt.Errorf("%s.%s: no such field", root, strings.Join(path[:i], "."))
        }
    }
    return v, nil
}
```

```go
// note the generic here: we want to be able to set any field on any struct,
// but to do so we need an addressable instance. by making sure the root is a pointer,
// non-nil root must be an addressable instance.
func exec[T any](pt *T, line string) (logMsg string, err error) {
    defer func() { 
        // the debug console should never crash the game.
        if p := recover(); p != nil {
            err = fmt.Errorf("panic: %v", p)
        }
    }
    root := reflect.ValueOf(pt).Elem() // get the reflect.Value of the dereference of pt
    // split the line on whitespace
    args := strings.Fields(line)
    op, args := args[0], args[1:]
    switch op {
    default:
        return "", fmt.Errorf("unknown op %s", op)
    case "print":
        if len(args) != 1 {
            return "", fmt.Errorf("print: expected 1 argument, got %d", len(args))
        }
        v, err := resolvePath(root, args[0])
        if err != nil {
            return "", err
        }
        return fmt.Sprintf("%s = %v", args[0], v), nil
    case "set":
        if len(args) != 2 {
            return "", fmt.Errorf("set: expected 2 arguments, got %d", len(args))
        }
        lhs, err := resolvePath(root, args[0])
        if err != nil {
            return "", err
        }
        rhs, err := resolvePath(root, args[1])
     
    }
}
```

- We want to be able to set variables without worrying too much about types. That is, we want to be "stringly typed":

-

```
set player.health 100
```

should succeed regardless of whether player.health is an int16 or float32.

- By default, we should be able to set any field on any struct, but we need some way to filter out fields that are too 'fragile' to be modified at runtime. Go already has a convention for this: fields that start with a lowercase letter are not exported, and cannot be accessed outside of the package. It turns out the reflect package respects this convention, and will not allow you to set an unexported field. Since the GameState struct is at the "root" of our program, in main, and nothing else will actually import it, we can just use capital letters to denote fields that are safe to modify at runtime.

```go
type Gamestate struct {
    Player struct {
        Health int
        Ammo int
        X, Y int
        dangerousField unsafe.Pointer // this field is not exported, and cannot be modified at runtime
    }
}
```
