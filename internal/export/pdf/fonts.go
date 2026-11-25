package pdf

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/signintech/gopdf/fontmaker/core"
)

// fontFiles includes the license and notice texts as well as the font binaries,
// so source and binary distributions retain the embedded-font provenance.
//
//go:embed assets/fonts
var fontFiles embed.FS

type fontAsset struct {
	family string
	style  string
	path   string
	url    string
}

// These URLs are cache keys, not runtime locations. They exactly match the
// URLs in go-swiss/fonts' embedded Roboto metadata at the pinned dependency
// versions. The renderer uses plain HTTP keys even though the assets were
// retrieved over HTTPS when vendored.
var fontAssets = []fontAsset{
	{
		family: "Roboto",
		style:  "",
		path:   "assets/fonts/Roboto-Regular.ttf",
		url:    "http://fonts.gstatic.com/s/roboto/v30/KFOmCnqEu92Fr1Me5WZLCzYlKw.ttf",
	},
	{
		family: "Roboto",
		style:  "I",
		path:   "assets/fonts/Roboto-Italic.ttf",
		url:    "http://fonts.gstatic.com/s/roboto/v30/KFOkCnqEu92Fr1Mu52xPKTM1K9nz.ttf",
	},
	{
		family: "Roboto",
		style:  "B",
		path:   "assets/fonts/Roboto-Bold.ttf",
		url:    "http://fonts.gstatic.com/s/roboto/v30/KFOlCnqEu92Fr1MmWUlvAx05IsDqlA.ttf",
	},
	{
		family: "Roboto",
		style:  "BI",
		path:   "assets/fonts/Roboto-BoldItalic.ttf",
		url:    "http://fonts.gstatic.com/s/roboto/v30/KFOjCnqEu92Fr1Mu51TzBhc9AMX6lJBP.ttf",
	},
	{
		family: "Roboto Mono",
		style:  "",
		path:   "assets/fonts/RobotoMono-Regular.ttf",
		url:    "http://fonts.gstatic.com/s/robotomono/v22/L0xuDF4xlVMF-BfR8bXMIhJHg45mwgGEFl0_3vqPQ--5Ip2sSQ.ttf",
	},
	{
		family: "Roboto Mono",
		style:  "I",
		path:   "assets/fonts/RobotoMono-Italic.ttf",
		url:    "http://fonts.gstatic.com/s/robotomono/v22/L0xoDF4xlVMF-BfR8bXMIjhOsXG-q2oeuFoqFrlnAOW9AJi8SZwt.ttf",
	},
	{
		family: "Roboto Mono",
		style:  "B",
		path:   "assets/fonts/RobotoMono-Bold.ttf",
		url:    "http://fonts.gstatic.com/s/robotomono/v22/L0xuDF4xlVMF-BfR8bXMIhJHg45mwgGEFl0_Of2PQ--5Ip2sSQ.ttf",
	},
	{
		family: "Roboto Mono",
		style:  "BI",
		path:   "assets/fonts/RobotoMono-BoldItalic.ttf",
		url:    "http://fonts.gstatic.com/s/robotomono/v22/L0xoDF4xlVMF-BfR8bXMIjhOsXG-q2oeuFoqFrmAB-W9AJi8SZwt.ttf",
	},
}

// offlineFontCache is read-only after New returns. Set intentionally does
// nothing. Unknown keys return a nil hit to stop go-swiss/fonts' HTTP fallback;
// the renderer then reports an invalid-font error instead of using the network.
type offlineFontCache struct {
	values    map[string][]byte
	coverage  map[fontKey]runeCoverage
	cacheHits atomic.Int64
}

func (cache *offlineFontCache) Get(key string) ([]byte, bool) {
	value, ok := cache.values[key]
	if ok {
		cache.cacheHits.Add(1)
		return value, true
	}
	return nil, true
}

func (cache *offlineFontCache) Set(string, []byte) {}

func loadOfflineFonts() (*offlineFontCache, error) {
	cache := &offlineFontCache{
		values:   make(map[string][]byte, len(fontAssets)),
		coverage: make(map[fontKey]runeCoverage, len(fontAssets)),
	}
	for _, asset := range fontAssets {
		data, err := fontFiles.ReadFile(asset.path)
		if err != nil {
			return nil, fmt.Errorf("read embedded font %s: %w", asset.path, err)
		}
		coverage, err := parseFontCoverage(data)
		if err != nil {
			return nil, fmt.Errorf("parse embedded font %s: %w", asset.path, err)
		}
		cache.values[asset.url] = data
		cache.coverage[fontKey{family: asset.family, style: normalizeFontStyle(asset.style)}] = coverage
	}
	return cache, nil
}

type runeCoverage map[int]uint

func parseFontCoverage(data []byte) (runeCoverage, error) {
	parser := &core.TTFParser{}
	if err := parser.ParseByReader(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return parser.Chars(), nil
}

type fontKey struct {
	family string
	style  string
}

func normalizeFontStyle(style string) string {
	return strings.ReplaceAll(strings.ToUpper(style), "U", "")
}
