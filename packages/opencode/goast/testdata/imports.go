package imports

import "fmt"

import "os"

import (
	"context"
	"io"
	"net/http"

	pb "google.golang.org/protobuf/proto"
	_ "net/http/pprof"
)

func UseImports() {
	fmt.Println("hello")
	_ = os.Stdin
	_ = context.Background()
	_ = io.EOF
	_ = http.StatusOK
	_ = pb.Marshal
}
