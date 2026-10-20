package updater

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// checkEvery is how long a cached answer is trusted before asking GitHub again.
const checkEvery = 24 * time.Hour

// Notifier answers "is there a newer version?" from a cache file, so the
// notice costs no network on most runs.
type Notifier struct {
	Client *Client
	Cache  string // version_check.json
	Now    func() time.Time
}

type cacheState struct {
	Checked int64  `json:"last_check"`
	Latest  string `json:"latest"`
}

func (n *Notifier) read() cacheState {
	var st cacheState
	if data, err := os.ReadFile(n.Cache); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	return st
}

// Remember stores latest as the answer of a check made now.
func (n *Notifier) Remember(latest string) {
	data, err := json.Marshal(cacheState{Checked: n.Now().Unix(), Latest: latest})
	if err != nil || os.MkdirAll(filepath.Dir(n.Cache), 0o700) != nil {
		return
	}
	_ = os.WriteFile(n.Cache, data, 0o600)
}

// Cached returns the newer version the cache knows of, or "".
func (n *Notifier) Cached(current string) string {
	if st := n.read(); IsNewer(st.Latest, current) {
		return st.Latest
	}
	return ""
}

// Stale reports whether the cache is missing or older than a day.
func (n *Notifier) Stale() bool {
	st := n.read()
	return st.Checked == 0 || n.Now().Sub(time.Unix(st.Checked, 0)) > checkEvery
}

// Refresh asks GitHub, caches the answer and returns the newer version, or "".
// A failed check leaves the cache as it was, so the next run retries.
func (n *Notifier) Refresh(ctx context.Context, current string) string {
	latest, err := n.Client.Latest(ctx)
	if err != nil {
		return ""
	}
	n.Remember(latest)
	if IsNewer(latest, current) {
		return latest
	}
	return ""
}
