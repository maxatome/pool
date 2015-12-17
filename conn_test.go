package pool

import (
	"testing"
)

func TestRconn_Impl(t *testing.T) {
	var _ RpcAble = new(PoolRconn)
}
