package plugin

import (
	"fmt"
	"strings"

	foundryproto "github.com/cafecito-games/foundry-tools/internal/proto"
)

// parseParameter turns protoc's --foundryscript_opt string into generation
// options. protoc joins repeated _opt flags with commas, so "json" and
// "json,json" both mean the same thing.
//
// An unrecognized key is an error rather than something to ignore: a misspelled
// option that silently does nothing produces output the caller did not ask for
// and has no way to notice.
func parseParameter(parameter string) (foundryproto.Options, error) {
	var options foundryproto.Options
	for _, entry := range strings.Split(parameter, ",") {
		switch key := strings.TrimSpace(entry); key {
		case "":
			continue
		case "json":
			options.JSON = true
		default:
			return foundryproto.Options{}, fmt.Errorf("unknown generator option %q", key)
		}
	}
	return options, nil
}
