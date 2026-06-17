package main

import (
	"fmt"
	"os"

	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

func main() {
	prefix := types.IDPrefix("tr") // 默认 trace 前缀
	if len(os.Args) > 1 {
		prefix = types.IDPrefix(os.Args[1])
	}
	fmt.Print(utils.GenerateULID(prefix))
}
