package backend

import (
	"net/url"

	"github.com/yudai/gotty/webtty"
)

type Factory interface {
	New(params url.Values) (webtty.Slave, error)
}
