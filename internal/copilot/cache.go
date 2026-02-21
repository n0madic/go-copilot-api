package copilot

import (
	"context"

	"github.com/n0madic/go-copilot-api/internal/state"
)

func CacheModels(ctx context.Context, client *Client, st *state.State) error {
	models, err := client.GetModels(ctx)
	if err != nil {
		return err
	}
	st.SetModels(&models)
	return nil
}
