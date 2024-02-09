package utils

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	adjectives = [10]string{
		"whimscal",
		"bubbly",
		"gluffy",
		"quirky",
		"merry",
		"zigzagging",
		"charming",
		"jubilant",
		"lively",
		"enchanted",
	}
	nouns = [10]string{
		"rainbow",
		"unicorn",
		"penguin",
		"banana",
		"jellybean",
		"giggles",
		"sunflower",
		"snickers",
		"marshmallow",
		"cupcake",
	}
)

func GetRandomNames() string {
	rand.NewSource(time.Now().UnixNano())
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%v-%v", adj, noun)
}
