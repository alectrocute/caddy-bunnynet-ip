package caddy_bunnynet_ip

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const (
	bunnyIPv4 = "https://api.bunny.net/mc/nodes/plain"
	// bunny.net currently doesn't support ipv6
	// but let's plan on someday supporting it
	bunnyIPv6 = ""
)

func init() {
	caddy.RegisterModule(BunnyIPRange{})
}

// BunnyIPRange provides a range of IP address prefixes (CIDRs) retrieved from bunny.net.
type BunnyIPRange struct {
	// refresh Interval
	Interval caddy.Duration `json:"interval,omitempty"`
	// request Timeout
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// Holds the parsed CIDR ranges from Ranges.
	ranges []netip.Prefix

	ctx  caddy.Context
	lock *sync.RWMutex
}

// CaddyModule returns the Caddy module information.
func (BunnyIPRange) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.ip_sources.bunnynet",
		New: func() caddy.Module { return new(BunnyIPRange) },
	}
}

// getContext returns a cancelable context, with a timeout if configured.
func (s *BunnyIPRange) getContext() (context.Context, context.CancelFunc) {
	if s.Timeout > 0 {
		return context.WithTimeout(s.ctx, time.Duration(s.Timeout))
	}
	return context.WithCancel(s.ctx)
}

func parseBunnyNode(line string) (netip.Prefix, error) {
	text := strings.TrimSpace(line)
	if text == "" {
		return netip.Prefix{}, fmt.Errorf("empty line")
	}

	// Keep compatibility with CIDR input, while supporting Bunny's plain IP format.
	prefix, err := caddyhttp.CIDRExpressionToPrefix(text)
	if err == nil {
		return prefix, nil
	}

	addr, parseErr := netip.ParseAddr(text)
	if parseErr != nil {
		return netip.Prefix{}, err
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

func (s *BunnyIPRange) fetch(api string) ([]netip.Prefix, error) {
	if strings.TrimSpace(api) == "" {
		return nil, nil
	}

	ctx, cancel := s.getContext()
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status from %s: %s", api, resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	var prefixes []netip.Prefix
	for scanner.Scan() {
		prefix, err := parseBunnyNode(scanner.Text())
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return prefixes, nil
}

func (s *BunnyIPRange) getPrefixes() ([]netip.Prefix, error) {
	var fullPrefixes []netip.Prefix
	// fetch ipv4 list
	prefixes, err := s.fetch(bunnyIPv4)
	if err != nil {
		return nil, err
	}
	fullPrefixes = append(fullPrefixes, prefixes...)

	// fetch ipv6 list
	prefixes, err = s.fetch(bunnyIPv6)
	if err != nil {
		return nil, err
	}
	fullPrefixes = append(fullPrefixes, prefixes...)

	return fullPrefixes, nil
}

func (s *BunnyIPRange) Provision(ctx caddy.Context) error {
	s.ctx = ctx
	s.lock = new(sync.RWMutex)

	// update in background
	go s.refreshLoop()
	return nil
}

func (s *BunnyIPRange) refreshLoop() {
	if s.Interval == 0 {
		s.Interval = caddy.Duration(time.Hour)
	}

	ticker := time.NewTicker(time.Duration(s.Interval))
	// first time update
	s.lock.Lock()
	// it's nil anyway if there is an error
	s.ranges, _ = s.getPrefixes()
	s.lock.Unlock()
	for {
		select {
		case <-ticker.C:
			fullPrefixes, err := s.getPrefixes()
			if err != nil {
				break
			}

			s.lock.Lock()
			s.ranges = fullPrefixes
			s.lock.Unlock()
		case <-s.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (s *BunnyIPRange) GetIPRanges(_ *http.Request) []netip.Prefix {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ranges
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
//	bunnynet {
//	   interval val
//	   timeout val
//	}
func (m *BunnyIPRange) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // Skip module name.

	// No same-line options are supported
	if d.NextArg() {
		return d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "interval":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return err
			}
			m.Interval = caddy.Duration(val)
		case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return err
			}
			m.Timeout = caddy.Duration(val)
		default:
			return d.ArgErr()
		}
	}

	return nil
}

// interface guards
var (
	_ caddy.Module            = (*BunnyIPRange)(nil)
	_ caddy.Provisioner       = (*BunnyIPRange)(nil)
	_ caddyfile.Unmarshaler   = (*BunnyIPRange)(nil)
	_ caddyhttp.IPRangeSource = (*BunnyIPRange)(nil)
)
