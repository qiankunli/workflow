package internal

import (
	"fmt"
	"testing"
)

func TestNewStep(t *testing.T) {
	_, ok := Factory["random"]
	fmt.Println(ok)
}
