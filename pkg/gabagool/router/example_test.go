package router_test

import (
	"fmt"

	"github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/router"
)

// Screen identifiers - use typed constants for compile-time safety
const (
	ScreenGameList router.Screen = iota
	ScreenGameDetail
	ScreenSettings
)

// Action enums for each screen
type GameListAction int

const (
	GameListActionSelected GameListAction = iota
	GameListActionSettings
	GameListActionBack
)

type GameDetailAction int

const (
	GameDetailActionBack GameDetailAction = iota
	GameDetailActionPlay
)

// Domain types
type Game struct {
	ID   int
	Name string
}

// Input types - what each screen needs to render
type GameListInput struct {
	Games  []Game
	Resume *GameListResume
}

type GameDetailInput struct {
	Game Game
}

// Result types - what each screen returns
type GameListResult struct {
	Action   GameListAction
	Selected *Game
	Resume   *GameListResume
}

type GameDetailResult struct {
	Action GameDetailAction
}

// Resume types - position state for back navigation
type GameListResume struct {
	SelectedIndex  int
	ScrollPosition int
}

// Example demonstrates basic router usage with screen registration and transitions.
func Example() {
	r := router.New()

	// Track calls to simulate a flow: list -> detail -> back -> exit
	listCalls := 0
	detailCalls := 0

	// Register screens
	r.Register(ScreenGameList, func(input any) (any, error) {
		in := input.(GameListInput)
		listCalls++

		if listCalls == 1 {
			// First call: select a game
			fmt.Println("List: selecting game")
			return GameListResult{
				Action:   GameListActionSelected,
				Selected: &in.Games[0],
				Resume:   &GameListResume{SelectedIndex: 0},
			}, nil
		}
		// Second call: exit
		fmt.Printf("List: restored to index %d, exiting\n", in.Resume.SelectedIndex)
		return GameListResult{Action: GameListActionBack}, nil
	})

	r.Register(ScreenGameDetail, func(input any) (any, error) {
		in := input.(GameDetailInput)
		detailCalls++
		fmt.Printf("Detail: showing %s, going back\n", in.Game.Name)
		return GameDetailResult{Action: GameDetailActionBack}, nil
	})

	// Define all transitions in one place
	r.OnTransition(func(from router.Screen, result any, stack *router.Stack) (router.Screen, any) {
		switch from {
		case ScreenGameList:
			res := result.(GameListResult)
			switch res.Action {
			case GameListActionSelected:
				// Forward: push current state, go to detail
				stack.Push(from, GameListInput{Games: []Game{{ID: 1, Name: "Portal"}}}, res.Resume)
				return ScreenGameDetail, GameDetailInput{Game: *res.Selected}
			case GameListActionBack:
				return router.ScreenExit, nil
			}

		case ScreenGameDetail:
			res := result.(GameDetailResult)
			if res.Action == GameDetailActionBack {
				// Back: pop and restore
				if entry := stack.Pop(); entry != nil {
					in := entry.Input.(GameListInput)
					if entry.Resume != nil {
						in.Resume = entry.Resume.(*GameListResume)
					}
					return entry.Screen, in
				}
			}
			return router.ScreenExit, nil
		}
		return router.ScreenExit, nil
	})

	// Start the router
	games := []Game{{ID: 1, Name: "Portal"}}
	_ = r.Run(ScreenGameList, GameListInput{Games: games})

	// Output:
	// List: selecting game
	// Detail: showing Portal, going back
	// List: restored to index 0, exiting
}

// Example_backNavigation demonstrates stack-based back navigation with resume state.
func Example_backNavigation() {
	r := router.New()

	visits := 0

	r.Register(ScreenGameList, func(input any) (any, error) {
		in := input.(GameListInput)
		visits++

		if visits == 1 {
			fmt.Println("First visit: selecting game at index 2")
			return GameListResult{
				Action:   GameListActionSelected,
				Selected: &Game{ID: 1, Name: "Half-Life"},
				Resume:   &GameListResume{SelectedIndex: 2, ScrollPosition: 100},
			}, nil
		}

		// Returning from detail
		fmt.Printf("Returned: index=%d, scroll=%d\n",
			in.Resume.SelectedIndex, in.Resume.ScrollPosition)
		return GameListResult{Action: GameListActionBack}, nil
	})

	r.Register(ScreenGameDetail, func(input any) (any, error) {
		in := input.(GameDetailInput)
		fmt.Printf("Viewing: %s\n", in.Game.Name)
		return GameDetailResult{Action: GameDetailActionBack}, nil
	})

	r.OnTransition(func(from router.Screen, result any, stack *router.Stack) (router.Screen, any) {
		switch from {
		case ScreenGameList:
			res := result.(GameListResult)
			if res.Action == GameListActionSelected {
				stack.Push(from, GameListInput{}, res.Resume)
				return ScreenGameDetail, GameDetailInput{Game: *res.Selected}
			}
			return router.ScreenExit, nil

		case ScreenGameDetail:
			if entry := stack.Pop(); entry != nil {
				in := entry.Input.(GameListInput)
				if entry.Resume != nil {
					in.Resume = entry.Resume.(*GameListResume)
				}
				return entry.Screen, in
			}
			return router.ScreenExit, nil
		}
		return router.ScreenExit, nil
	})

	_ = r.Run(ScreenGameList, GameListInput{})

	// Output:
	// First visit: selecting game at index 2
	// Viewing: Half-Life
	// Returned: index=2, scroll=100
}
